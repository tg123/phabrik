package transport

import (
	"bytes"
	"io"
	"sync/atomic"

	"github.com/tg123/phabrik/serialization"
)

type Message struct {
	Headers MessageHeaders
	Body    interface{}
}

type ByteArrayMessage struct {
	Headers MessageHeaders
	Body    []byte
}

func (m *Message) marshal() (int, []byte, error) {
	var buf bytes.Buffer

	err := m.Headers.writeTo(&buf)
	if err != nil {
		return 0, nil, err
	}

	headerLen := buf.Len()

	if m.Body != nil {
		b, ok := m.Body.([]byte)

		if !ok {
			b, err = serialization.Marshal(m.Body)
			if err != nil {
				return 0, nil, err
			}
		}

		_, err = buf.Write(b)
		if err != nil {
			return 0, nil, err
		}
	}

	return headerLen, buf.Bytes(), nil
}

func writeMessageWithFrame(w io.Writer, message *Message, config frameWriteConfig) error {
	headerLen, msg, err := message.marshal()
	if err != nil {
		return err
	}

	return writeFrame(w, headerLen, msg, config)
}

func nextMessageHeaderAndBodyFromFrame(r io.Reader, config frameReadConfig) (*MessageHeaders, []byte, error) {
	frameheader, framebody, err := nextFrame(r, config)
	if err != nil {
		return nil, nil, err
	}

	headers, err := parseFabricMessageHeaders(bytes.NewBuffer(framebody[:frameheader.HeaderLength]))
	if err != nil {
		return nil, nil, err
	}

	body := framebody[frameheader.HeaderLength:]
	return headers, body, nil
}

type messageFactory struct {
	messagePrefix serialization.GUID
	messageIdx    uint32
}

func newMessageFactory() (*messageFactory, error) {
	g, err := serialization.NewGuidV4()
	if err != nil {
		return nil, err
	}

	return &messageFactory{g, 0}, nil
}

func (f *messageFactory) newMessage() *Message {
	msg := &Message{}
	msg.Headers.customHeaders = make(map[MessageHeaderIdType][]interface{})
	f.fillMessageId(msg)

	return msg
}

func (f *messageFactory) fillMessageId(message *Message) {
	if message.Headers.Id.IsEmpty() {
		message.Headers.Id = f.Next()
	}
}

func (f *messageFactory) Next() MessageId {
	return MessageId{f.messagePrefix, atomic.AddUint32(&f.messageIdx, 1)}
}

type MessageIdGenerator interface {
	Next() MessageId
}

func NewMessageIdGenerator() (MessageIdGenerator, error) {
	return newMessageFactory()
}
