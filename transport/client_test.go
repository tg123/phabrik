package transport

import (
	"bytes"
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

// https://github.com/golang/crypto/blob/38f3c27a63bf8d9928ce230b01cab346d1756e88/ssh/handshake_test.go#L42
// netPipe is analogous to net.Pipe, but it uses a real net.Conn, and
// therefore is buffered (net.Pipe deadlocks if both sides start with
// a write.)
func netPipe() (net.Conn, net.Conn, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		listener, err = net.Listen("tcp", "[::1]:0")
		if err != nil {
			return nil, nil, err
		}
	}
	defer listener.Close()
	c1, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		return nil, nil, err
	}

	c2, err := listener.Accept()
	if err != nil {
		c1.Close()
		return nil, nil, err
	}

	return c1, c2, nil
}

func TestBasicMessage(t *testing.T) {
	p1, p2, err := netPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer p1.Close()
	defer p2.Close()

	c1, err := Connect(p1, Config{})
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		msg := &Message{}
		msg.Headers.Action = "TEST"
		msg.Headers.Actor = MessageActorTypeGenericTestActor
		msg.Body = []byte{1, 2, 3, 4}
		err := c1.SendOneWay(context.Background(), msg)
		if err != nil {
			t.Error(err)
		}
	}()

	frameheader, framebody, err := nextFrame(p2, c1.frameRCfg)
	if err != nil {
		t.Fatal(err)
	}

	headers, err := parseFabricMessageHeaders(bytes.NewBuffer(framebody[:frameheader.HeaderLength]))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "TEST", headers.Action)
	assert.Equal(t, MessageActorTypeGenericTestActor, headers.Actor)

	body := framebody[frameheader.HeaderLength:]
	assert.Equal(t, []byte{1, 2, 3, 4}, body)
}

func TestRequestMessage(t *testing.T) {
	p1, p2, err := netPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer p1.Close()
	defer p2.Close()

	c, err := Connect(p1, Config{})
	if err != nil {
		t.Fatal(err)
	}

	go c.Run(context.Background())

	// fake server by just handler
	s, err := Connect(p2, Config{
		MessageHandlers: map[MessageActorType]MessageHandler{
			MessageActorTypeGenericTestActor: func(client *Client, bam *ByteArrayMessage) {

				assert.Equal(t, "TEST", bam.Headers.Action)
				assert.Equal(t, MessageActorTypeGenericTestActor, bam.Headers.Actor)
				assert.Equal(t, []byte{1, 2, 3, 4}, bam.Body)

				msg := &Message{}
				msg.Headers.RelatesTo = bam.Headers.Id
				msg.Headers.Action = "TEST_REPLY"
				msg.Headers.Actor = MessageActorTypeGenericTestActor
				msg.Body = []byte{4, 3, 2, 1}
				err := client.SendOneWay(context.Background(), msg)
				if err != nil {
					t.Fatal(err)
				}
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	go s.Run(context.Background())

	{
		msg := &Message{}
		msg.Headers.Action = "TEST"
		msg.Headers.Actor = MessageActorTypeGenericTestActor
		msg.Body = []byte{1, 2, 3, 4}
		reply, err := c.RequestReply(context.Background(), msg)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, "TEST_REPLY", reply.Headers.Action)
		assert.Equal(t, MessageActorTypeGenericTestActor, reply.Headers.Actor)
		assert.Equal(t, []byte{4, 3, 2, 1}, reply.Body)
	}
}

func TestMessageCRC(t *testing.T) {
	var buf bytes.Buffer

	err := writeMessageWithFrame(&buf, &Message{
		Body: []byte("string"),
	}, frameWriteConfig{
		FrameHeaderCRC: true,
		FrameBodyCRC:   true,
	})

	if err != nil {
		t.Fatal(err)
	}

	data := buf.Bytes()

	t.Run("should not fail", func(t *testing.T) {
		_, _, err = nextFrame(bytes.NewBuffer(data), frameReadConfig{
			CheckFrameHeaderCRC: true,
			CheckFrameBodyCRC:   true,
		})
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("should fail with header mess", func(t *testing.T) {
		data2 := append(data[:0:0], data...)

		// mess header
		data2[5] = '2'

		_, _, err = nextFrame(bytes.NewBuffer(data2), frameReadConfig{
			CheckFrameHeaderCRC: true,
			CheckFrameBodyCRC:   false,
		})

		if err == nil {
			t.Error("should fail but no error")
		}
	})

	t.Run("should fail with body mess", func(t *testing.T) {
		data2 := append(data[:0:0], data...)

		// mess data
		data2[len(data2)-1] = '1'

		_, _, err = nextFrame(bytes.NewBuffer(data2), frameReadConfig{
			CheckFrameHeaderCRC: false,
			CheckFrameBodyCRC:   true,
		})

		if err == nil {
			t.Error("should fail but no error")
		}
	})

	t.Run("should not fail without header crc", func(t *testing.T) {
		data2 := append(data[:0:0], data...)

		// mess data
		data2[5] = '2'

		_, _, err = nextFrame(bytes.NewBuffer(data2), frameReadConfig{
			CheckFrameHeaderCRC: false,
			CheckFrameBodyCRC:   true,
		})

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("should not fail without body crc", func(t *testing.T) {
		data2 := append(data[:0:0], data...)

		// mess data
		data2[len(data2)-1] = '1'

		_, _, err = nextFrame(bytes.NewBuffer(data2), frameReadConfig{
			CheckFrameHeaderCRC: true,
			CheckFrameBodyCRC:   false,
		})

		if err != nil {
			t.Error(err)
		}
	})

}
