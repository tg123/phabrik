package transport

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/tg123/phabrik/serialization"
)

type MessageId struct {
	Id    serialization.GUID
	Index uint32
}

type ActivityId MessageId

func (m MessageId) IsEmpty() bool {
	return m.Id.IsEmpty() && m.Index == 0
}

func (m MessageId) String() string {
	return fmt.Sprintf("%v:%v", m.Id.String(), m.Index)
}

// var _ serialization.CustomMarshaler = (*MessageId)(nil)

// func (r *MessageId) Marshal(s serialization.Encoder) error {
// 	return nil
// }

// func (r *MessageId) Unmarshal(meta serialization.FabricSerializationType, s serialization.Decoder) error {
// 	return nil
// }

type MessageHeaders struct {
	Id        MessageId
	RelatesTo MessageId

	Actor  MessageActorType
	Action string

	ExpectsReply bool
	HighPriority bool
	Idempotent   bool

	ErrorCode    FabricErrorCode
	HasFaultBody bool
	RetryCount   int32

	customHeaders map[MessageHeaderIdType][]interface{}
}

type listenInstance struct {
	Address                string
	Instance               uint64
	Nonce                  serialization.GUID
	HeartbeatSupported     bool
	ConnectionFeatureFlags uint32
}

type securityNegotiationHeader struct {
	X509ExtraFramingEnabled  bool
	FramingProtectionEnabled bool
	ListenInstance           listenInstance
	MaxIncomingFrameSize     uint64
}

type HeaderTypeActivator func() interface{}

var headerTypeActivators = map[MessageHeaderIdType]HeaderTypeActivator{}

func RegisterHeaderActivator(typ MessageHeaderIdType, activator HeaderTypeActivator) {
	headerTypeActivators[typ] = activator
}

func (h *MessageHeaders) GetCustomHeaders(typ MessageHeaderIdType) []interface{} {
	return h.customHeaders[typ]
}

func (h *MessageHeaders) GetFirstCustomHeader(typ MessageHeaderIdType) (interface{}, bool) {
	headers := h.customHeaders[typ]

	if len(headers) > 0 {
		return headers[0], true
	}

	return nil, false
}

func (h *MessageHeaders) SetCustomHeader(typ MessageHeaderIdType, header interface{}) bool {
	if h.customHeaders == nil {
		h.customHeaders = make(map[MessageHeaderIdType][]interface{})
	}

	if _, ok := h.customHeaders[typ]; !ok {
		h.customHeaders[typ] = []interface{}{header}
		return true
	}

	return false
}

func (h *MessageHeaders) AppendCustomHeader(typ MessageHeaderIdType, header ...interface{}) bool {
	if h.customHeaders == nil {
		h.customHeaders = make(map[MessageHeaderIdType][]interface{})
	}

	if _, ok := h.customHeaders[typ]; !ok {
		h.customHeaders[typ] = append(h.customHeaders[typ], header...)
		return true
	}

	return false
}

func (h *MessageHeaders) writeTo(w io.Writer) error {

	if err := writeMessageHeader(w, MessageHeaderIdTypeMessageId, &h.Id); err != nil {
		return err
	}

	if !h.RelatesTo.IsEmpty() {
		if err := writeMessageHeader(w, MessageHeaderIdTypeRelatesTo, &h.RelatesTo); err != nil {
			return err
		}
	}

	if err := writeMessageHeader(w, MessageHeaderIdTypeActor, &struct {
		Actor MessageActorType
	}{
		Actor: h.Actor,
	}); err != nil {
		return err
	}

	if h.Action != "" {
		if err := writeMessageHeader(w, MessageHeaderIdTypeAction, &struct {
			Action string
		}{
			Action: h.Action,
		}); err != nil {
			return err
		}
	}

	if h.ExpectsReply {
		if err := writeMessageHeader(w, MessageHeaderIdTypeExpectsReply, &struct {
			ExpectsReply bool
		}{
			ExpectsReply: h.ExpectsReply,
		}); err != nil {
			return err
		}
	}

	if h.HighPriority {
		if err := writeMessageHeader(w, MessageHeaderIdTypeHighPriority, &struct {
			HighPriority bool
		}{
			HighPriority: h.HighPriority,
		}); err != nil {
			return err
		}
	}

	if h.Idempotent {
		if err := writeMessageHeader(w, MessageHeaderIdTypeIdempotent, &struct {
			Idempotent bool
		}{
			Idempotent: h.Idempotent,
		}); err != nil {
			return err
		}
	}

	if h.ErrorCode != 0 || h.HasFaultBody {
		if err := writeMessageHeader(w, MessageHeaderIdTypeFault, &struct {
			ErrorCode    FabricErrorCode
			HasFaultBody bool
		}{
			ErrorCode:    h.ErrorCode,
			HasFaultBody: h.HasFaultBody,
		}); err != nil {
			return err
		}
	}

	if h.RetryCount > 0 {
		if err := writeMessageHeader(w, MessageHeaderIdTypeRetry, &struct {
			RetryCount int32
		}{
			RetryCount: h.RetryCount,
		}); err != nil {
			return err
		}
	}

	for k, v := range h.customHeaders {
		for _, ch := range v {
			if err := writeMessageHeader(w, k, ch); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeMessageHeader(w io.Writer, id MessageHeaderIdType, headerbody interface{}) error {
	var err error
	if err := binary.Write(w, binary.LittleEndian, id); err != nil {
		return err
	}

	b, ok := headerbody.([]byte)
	if !ok {
		b, err = serialization.Marshal(headerbody)
		if err != nil {
			return err
		}
	}

	if err := binary.Write(w, binary.LittleEndian, uint16(len(b))); err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

func nextMessageHeader(stream io.Reader) (MessageHeaderIdType, []byte, error) {
	var id MessageHeaderIdType

	err := binary.Read(stream, binary.LittleEndian, &id)
	if err != nil {
		return MessageHeaderIdTypeInvalid, nil, err
	}

	var size uint16
	err = binary.Read(stream, binary.LittleEndian, &size)
	if err != nil {
		return MessageHeaderIdTypeInvalid, nil, err
	}

	body := make([]byte, size)

	_, err = io.ReadFull(stream, body)
	if err != nil {
		return MessageHeaderIdTypeInvalid, nil, err
	}

	return id, body, nil
}

func parseFabricMessageHeaders(r io.Reader) (*MessageHeaders, error) {

	headers := MessageHeaders{
		customHeaders: make(map[MessageHeaderIdType][]interface{}),
	}

	for {

		id, headerdata, err := nextMessageHeader(r)
		if err == io.EOF {
			return &headers, nil
		}

		if err != nil {
			return nil, err
		}

		if id == MessageHeaderIdTypeInvalid {
			continue
		}

		switch id {
		case MessageHeaderIdTypeMessageId:
			if err := serialization.Unmarshal(headerdata, &headers.Id); err != nil {
				return nil, err
			}
		case MessageHeaderIdTypeRelatesTo:
			if err := serialization.Unmarshal(headerdata, &headers.RelatesTo); err != nil {
				return nil, err
			}
		case MessageHeaderIdTypeActor:
			var hv struct {
				Actor MessageActorType
			}
			if err := serialization.Unmarshal(headerdata, &hv); err != nil {
				return nil, err
			}

			headers.Actor = hv.Actor
		case MessageHeaderIdTypeAction:
			var hv struct {
				Action string
			}
			if err := serialization.Unmarshal(headerdata, &hv); err != nil {
				return nil, err
			}

			headers.Action = hv.Action
		case MessageHeaderIdTypeExpectsReply:
			var hv struct {
				ExpectsReply bool
			}
			if err := serialization.Unmarshal(headerdata, &hv); err != nil {
				return nil, err
			}

			headers.ExpectsReply = hv.ExpectsReply
		case MessageHeaderIdTypeHighPriority:
			var hv struct {
				HighPriority bool
			}
			if err := serialization.Unmarshal(headerdata, &hv); err != nil {
				return nil, err
			}

			headers.HighPriority = hv.HighPriority
		case MessageHeaderIdTypeIdempotent:
			var hv struct {
				Idempotent bool
			}
			if err := serialization.Unmarshal(headerdata, &hv); err != nil {
				return nil, err
			}

			headers.Idempotent = hv.Idempotent

		case MessageHeaderIdTypeSecurityNegotiation:
			// TODO support negotiation
		case MessageHeaderIdTypeFault:
			var hv struct {
				ErrorCode    FabricErrorCode
				HasFaultBody bool
			}

			if err := serialization.Unmarshal(headerdata, &hv); err != nil {
				return nil, err
			}

			headers.ErrorCode = hv.ErrorCode
			headers.HasFaultBody = hv.HasFaultBody

		case MessageHeaderIdTypeRetry:
			var hv struct {
				RetryCount int32
			}

			if err := serialization.Unmarshal(headerdata, &hv); err != nil {
				return nil, err
			}

			headers.RetryCount = hv.RetryCount
		default:

			activator := headerTypeActivators[id]

			if activator != nil {
				hv := activator()
				if err := serialization.Unmarshal(headerdata, hv); err != nil {
					return nil, err
				}

				headers.customHeaders[id] = append(headers.customHeaders[id], hv)
			} else {
				headers.customHeaders[id] = append(headers.customHeaders[id], headerdata)
			}
		}
	}

}
