package transport

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tg123/phabrik/serialization"
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

func mustTestConnection(t *testing.T, conn net.Conn) *connection {
	c, err := newConnection()
	if err != nil {
		t.Fatal(err)
	}

	c.conn = conn
	return c
}

func TestBasicMessage(t *testing.T) {
	p1, p2, err := netPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer p1.Close()
	defer p2.Close()

	c1 := mustTestConnection(t, p1)

	go func() {
		msg := &Message{}
		msg.Headers.Action = "TEST"
		msg.Headers.Actor = MessageActorTypeGenericTestActor
		msg.Body = []byte{1, 2, 3, 4}
		err := c1.SendOneWay(msg)
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

	go c.Wait()

	// fake server by just handler
	s, err := Connect(p2, Config{
		MessageCallback: func(client Conn, bam *ByteArrayMessage) {
			if bam.Headers.Actor != MessageActorTypeGenericTestActor {
				return
			}

			assert.Equal(t, "TEST", bam.Headers.Action)
			assert.Equal(t, []byte{1, 2, 3, 4}, bam.Body)

			msg := &Message{}
			msg.Headers.RelatesTo = bam.Headers.Id
			msg.Headers.Action = "TEST_REPLY"
			msg.Headers.Actor = MessageActorTypeGenericTestActor
			msg.Body = []byte{4, 3, 2, 1}
			err := client.SendOneWay(msg)
			if err != nil {
				t.Fatal(err)
			}
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	go s.Wait()

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

func TestCancelRequest(t *testing.T) {
	p1, p2, err := netPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer p1.Close()
	defer p2.Close()

	c := mustTestConnection(t, p1)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan int)
	go func() {
		st := time.Now()
		_, err := c.RequestReply(ctx, &Message{})
		if err != ctx.Err() {
			t.Error("not cancel error")
		} else if err == nil {
			t.Errorf("should return err")
		}

		if time.Since(st) < 1*time.Second {
			t.Errorf("should wait at least 1s")
		}

		done <- 1
	}()

	time.Sleep(1 * time.Second)
	cancel()

	<-done
}

func TestTransportMessages(t *testing.T) {
	p1, p2, err := netPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer p1.Close()
	defer p2.Close()

	c0, err := Connect(p1, Config{})
	if err != nil {
		t.Fatal(err)
	}

	go c0.Wait()

	c := c0.Conn.(*connection)

	t.Run("check init msg", func(t *testing.T) {

		frameheader, framebody, err := nextFrame(p2, c.frameRCfg)
		if err != nil {
			t.Fatal(err)
		}

		headers, err := parseFabricMessageHeaders(bytes.NewBuffer(framebody[:frameheader.HeaderLength]))
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, MessageActorTypeTransport, headers.Actor)

		b := transportInitMessageBody{}
		if err := serialization.Unmarshal(framebody[frameheader.HeaderLength:], &b); err != nil {
			t.Fatal(err)
		}

		// default flags
		assert.Equal(t, true, b.HeartbeatSupported)
		assert.Equal(t, uint32(1), b.ConnectionFeatureFlags)
	})

	go func() {
		var b struct {
			HeartbeatTimeTick int64
		}

		b.HeartbeatTimeTick = 1234567

		msg := c.msgfac.newMessage()
		msg.Headers.Actor = MessageActorTypeTransport
		msg.Headers.HighPriority = true
		msg.Headers.Action = "HeartbeatRequest"
		msg.Body = &b

		if err := writeMessageWithFrame(p2, msg, c.frameWCfg); err != nil {
			t.Error(err)
		}
	}()

	t.Run("check heartbeat msg", func(t *testing.T) {

		frameheader, framebody, err := nextFrame(p2, c.frameRCfg)
		if err != nil {
			t.Fatal(err)
		}

		headers, err := parseFabricMessageHeaders(bytes.NewBuffer(framebody[:frameheader.HeaderLength]))
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, MessageActorTypeTransport, headers.Actor)
		assert.Equal(t, "HeartbeatResponse", headers.Action)

		var b struct {
			HeartbeatTimeTick int64
		}

		if err := serialization.Unmarshal(framebody[frameheader.HeaderLength:], &b); err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, int64(1234567), b.HeartbeatTimeTick)
	})
}
