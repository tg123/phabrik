package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/tg123/phabrik/serialization"
)

type MessageCallback func(Conn, *ByteArrayMessage)

type MessageHandler interface {
	SetMessageCallback(actor MessageActorType, h MessageCallback)
}

type Config struct {
	MessageCallbacks map[MessageActorType]MessageCallback
	TLS              *tls.Config
	FrameHeaderCRC   bool
	FrameBodyCRC     bool
}

type Conn interface {
	RequestReply(ctx context.Context, message *Message) (*ByteArrayMessage, error)

	SendOneWay(message *Message) error

	Wait() error

	Close() error
}

type connection struct {
	handlerLock      sync.Mutex
	messageCallbacks map[MessageActorType]MessageCallback
	conn             net.Conn
	requestTable     sync.Map
	msgfac           *messageFactory

	frameRCfg frameReadConfig
	frameWCfg frameWriteConfig

	closed bool
}

func newConnection() (*connection, error) {
	mf, err := newMessageFactory()
	if err != nil {
		return nil, err
	}

	c := &connection{
		msgfac:           mf,
		messageCallbacks: make(map[MessageActorType]MessageCallback),
	}

	c.frameWCfg.SecurityProviderMask = securityProviderNone
	c.frameRCfg.CheckFrameHeaderCRC = true
	c.frameWCfg.FrameHeaderCRC = true

	return c, nil
}

func (c *connection) initHandlers(handlers map[MessageActorType]MessageCallback) {
	for k, v := range handlers {
		c.messageCallbacks[k] = v
	}
}

func (c *connection) setTls() {
	c.frameRCfg.CheckFrameHeaderCRC = false
	c.frameRCfg.CheckFrameBodyCRC = false
	c.frameWCfg.FrameHeaderCRC = false
	c.frameWCfg.FrameBodyCRC = false
	c.frameWCfg.SecurityProviderMask = securityProviderSsl
}

func (c *connection) SetMessageCallback(actor MessageActorType, h MessageCallback) {
	c.handlerLock.Lock()
	defer c.handlerLock.Unlock()

	if h == nil {
		delete(c.messageCallbacks, actor)
	} else {
		c.messageCallbacks[actor] = h
	}
}

func (c *connection) Close() error {
	c.requestTable.Range(func(key, value interface{}) bool {
		if ch, ok := value.(chan *ByteArrayMessage); ok {
			ch <- nil
		}

		return true
	})

	return c.conn.Close()
}

func (c *connection) handleTransportMessage(msg *ByteArrayMessage) error {
	switch msg.Headers.Action {
	case "HeartbeatRequest":
		var b struct {
			HeartbeatTimeTick int64
		}

		if err := serialization.Unmarshal(msg.Body, &b); err != nil {
			return err
		}

		{
			msg := c.msgfac.newMessage()
			msg.Headers.Actor = MessageActorTypeTransport
			msg.Headers.HighPriority = true
			msg.Headers.Action = "HeartbeatResponse"
			msg.Body = &b

			err := c.SendOneWay(msg)
			if err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

type transportInitMessageBody struct {
	Address                string
	Instance               uint64
	Nonce                  serialization.GUID
	HeartbeatSupported     bool
	ConnectionFeatureFlags uint32
}

func (c *connection) sendTransportInit(b *transportInitMessageBody) error {
	msg := c.msgfac.newMessage()
	msg.Headers.Actor = MessageActorTypeTransport
	msg.Headers.HighPriority = true
	msg.Body = b
	err := c.SendOneWay(msg)
	if err != nil {
		return err
	}

	return nil
}

func (c *connection) Wait() error {
	defer c.Close()

	for {
		frameheader, framebody, err := nextFrame(c.conn, c.frameRCfg)
		if err != nil {
			return err
		}

		headers, err := parseFabricMessageHeaders(bytes.NewBuffer(framebody[:frameheader.HeaderLength]))
		if err != nil {
			return err
		}

		body := framebody[frameheader.HeaderLength:]

		msg := &ByteArrayMessage{
			Headers: *headers,
			Body:    body,
		}

		if msg.Headers.Actor == MessageActorTypeEntreeServiceTransport {
			go c.handleTransportMessage(msg)
			continue
		}

		if !headers.RelatesTo.IsEmpty() {
			ch, ok := c.requestTable.LoadAndDelete(headers.RelatesTo.String())

			if ok {
				ch.(chan *ByteArrayMessage) <- msg
			} else {
				log.Printf("unknown reply %v", headers.RelatesTo)
			}
		} else {
			c.handlerLock.Lock()
			if h, ok := c.messageCallbacks[headers.Actor]; ok {
				go h(c, msg)
			}
			c.handlerLock.Unlock()
		}
	}
}

func (c *connection) SendOneWay(message *Message) error {
	c.msgfac.fillMessageId(message)
	return writeMessageWithFrame(c.conn, message, c.frameWCfg)
}

func (c *connection) RequestReply(ctx context.Context, message *Message) (*ByteArrayMessage, error) {
	c.msgfac.fillMessageId(message)
	message.Headers.ExpectsReply = true
	id := message.Headers.Id.String()
	defer c.requestTable.Delete(id)

	ch := make(chan *ByteArrayMessage)
	c.requestTable.Store(id, ch)

	err := writeMessageWithFrame(c.conn, message, c.frameWCfg)

	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case reply := <-ch:
		if reply == nil {
			return nil, fmt.Errorf("operation cancelled")
		}
		return reply, nil
	}

}