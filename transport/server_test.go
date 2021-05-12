package transport

import (
	"context"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicServer(t *testing.T) {
	server, err := ListenTCP("127.0.0.1:0", Config{
		MessageCallback: func(c Conn, bam *ByteArrayMessage) {
			msg := &Message{}
			msg.Headers.RelatesTo = bam.Headers.Id
			msg.Body = []byte(hex.EncodeToString(bam.Body))

			err := c.SendOneWay(msg)
			if err != nil {
				t.Error(err)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer server.Close()
	go server.Serve()

	t.Run("request reply", func(t *testing.T) {
		client, err := DialTCP(server.listener.Addr().String(), Config{})
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		go client.Wait()

		reply, err := client.RequestReply(context.TODO(), &Message{
			Body: []byte{1, 2, 3, 4, 5, 6},
		})

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, hex.EncodeToString([]byte{1, 2, 3, 4, 5, 6}), string(reply.Body))
	})

	t.Run("connect again", func(t *testing.T) {
		client, err := DialTCP(server.listener.Addr().String(), Config{})
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		go client.Wait()

		reply, err := client.RequestReply(context.TODO(), &Message{
			Body: []byte{6, 5, 4, 3, 2, 1},
		})

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, hex.EncodeToString([]byte{6, 5, 4, 3, 2, 1}), string(reply.Body))
	})
}
