package transport

import (
	"bytes"
	"sync/atomic"

	"github.com/tg123/phabrik/serialization"
)

type Message struct {
	Headers MessageHeaders
	Body    interface{}
}

func (m *Message) marshal() (int, []byte, error) {
	var buf bytes.Buffer

	err := m.Headers.writeTo(&buf)
	if err != nil {
		return 0, nil, err
	}

	headerLen := buf.Len()

	if m.Body != nil {
		b, err := serialization.Marshal(m.Body)
		if err != nil {
			return 0, nil, err
		}

		_, err = buf.Write(b)
		if err != nil {
			return 0, nil, err
		}
	}

	return headerLen, buf.Bytes(), nil
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
	msg.Headers.customHeaders = make(map[MessageHeaderIdType]interface{})
	f.fillMessageId(msg)

	return msg
}

func (f *messageFactory) fillMessageId(message *Message) {
	if message.Headers.Id.IsEmpty() {
		message.Headers.Id = MessageId{f.messagePrefix, atomic.AddUint32(&f.messageIdx, 1)}
	}
}
