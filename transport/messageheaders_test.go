package transport

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tg123/phabrik/serialization"
)

func TestMessageHeadersSerialization(t *testing.T) {
	var buf bytes.Buffer

	type timeoutHeader struct {
		Timeout time.Duration
	}
	RegisterHeaderActivator(MessageHeaderIdTypeTimeout, func() interface{} { return &timeoutHeader{} })

	var h MessageHeaders
	h.Action = "AC"
	h.Actor = MessageActorTypeGenericTestActor2
	h.ErrorCode = 100
	h.ExpectsReply = true
	h.HasFaultBody = true
	h.HighPriority = true
	h.Id = MessageId{serialization.MustNewGuidV4(), 100}
	h.Idempotent = true
	h.RelatesTo = MessageId{serialization.MustNewGuidV4(), 200}
	h.RetryCount = 4567
	h.SetCustomHeader(MessageHeaderIdTypeTimeout, &timeoutHeader{
		Timeout: 20 * time.Second,
	})

	err := h.writeTo(&buf)
	if err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()
	h2, err := parseFabricMessageHeaders(bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, h, *h2)

	{
		th, ok := h2.GetFirstCustomHeader(MessageHeaderIdTypeTimeout)
		assert.True(t, ok)
		assert.Equal(t, &timeoutHeader{Timeout: 20 * time.Second}, th)
	}

	{
		th, ok := h2.GetFirstCustomHeader(MessageHeaderIdTypeCustomClientAuth)
		assert.False(t, ok)
		assert.Equal(t, nil, th)

	}
}
