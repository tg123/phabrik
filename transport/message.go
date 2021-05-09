package transport

import (
	"bytes"

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
