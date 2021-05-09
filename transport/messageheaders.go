package transport

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/go-ole/go-ole"
	"github.com/tg123/phabrik/serialization"
)

type MessageId struct {
	Id    serialization.GUID
	Index uint32
}

func (m MessageId) IsEmpty() bool {
	return m.Id.IsEmpty() && m.Index == 0
}

func (m MessageId) String() string {
	g := ole.GUID(m.Id)
	return fmt.Sprintf("%v:%v", g.String(), m.Index)
}

type MessageHeaders struct {
	Id        MessageId
	RelatesTo MessageId

	Actor  MessageActorType
	Action string

	ExpectsReply bool
	HighPriority bool
	Idempotent   bool

	ErrorCode    serialization.FabricErrorCode
	HasFaultBody bool
	RetryCount   int32

	customHeaders map[MessageHeaderIdType]interface{}
}

type ListenInstance struct {
	Address                string
	Instance               uint64
	Nonce                  serialization.GUID
	HeartbeatSupported     bool
	ConnectionFeatureFlags uint32
}

type SecurityNegotiationHeader struct {
	X509ExtraFramingEnabled  bool
	FramingProtectionEnabled bool
	ListenInstance           ListenInstance
	MaxIncomingFrameSize     uint64
}

type HeaderTypeActivator func() interface{}

var headerTypeActivators = map[MessageHeaderIdType]HeaderTypeActivator{}

func RegisterHeaderActivator(typ MessageHeaderIdType, activator HeaderTypeActivator) {
	headerTypeActivators[typ] = activator
}

func (h *MessageHeaders) GetCustomHeader(typ MessageHeaderIdType) (interface{}, bool) {
	header, ok := h.customHeaders[typ]
	return header, ok
}

func (h *MessageHeaders) SetCustomHeader(typ MessageHeaderIdType, header interface{}) bool {
	if h.customHeaders == nil {
		h.customHeaders = make(map[MessageHeaderIdType]interface{})
	}

	if _, ok := h.customHeaders[typ]; !ok {
		h.customHeaders[typ] = header
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
		if err := writeMessageHeader(w, MessageHeaderIdTypeIdempotent, &struct {
			ErrorCode    serialization.FabricErrorCode
			HasFaultBody bool
		}{
			ErrorCode:    h.ErrorCode,
			HasFaultBody: h.HasFaultBody,
		}); err != nil {
			return err
		}
	}

	if h.RetryCount > 0 {
		if err := writeMessageHeader(w, MessageHeaderIdTypeIdempotent, &struct {
			RetryCount int32
		}{
			RetryCount: h.RetryCount,
		}); err != nil {
			return err
		}
	}

	for k, v := range h.customHeaders {
		if err := writeMessageHeader(w, k, v); err != nil {
			return err
		}
	}

	return nil
}

func writeMessageHeader(w io.Writer, id MessageHeaderIdType, headerbody interface{}) error {
	if err := binary.Write(w, binary.LittleEndian, id); err != nil {
		return err
	}

	b, err := serialization.Marshal(headerbody)
	if err != nil {
		return err
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

func parseFabricMessageHeaders(stream io.Reader) (*MessageHeaders, error) {

	headers := MessageHeaders{
		customHeaders: make(map[MessageHeaderIdType]interface{}),
	}

	for {

		id, headerdata, err := nextMessageHeader(stream)
		if err == io.EOF {
			return &headers, nil
		}

		if err != nil {
			return nil, err
		}

		if id == MessageHeaderIdTypeInvalid {
			continue
		}

		// only get first in c++ side
		if _, ok := headers.customHeaders[id]; ok {
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
				ErrorCode    serialization.FabricErrorCode
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
				if err := serialization.Unmarshal(headerdata, &hv); err != nil {
					return nil, err
				}

				headers.customHeaders[id] = hv
			}

			// log.Printf("unsupported msg header %v", id)
		}
	}

}
