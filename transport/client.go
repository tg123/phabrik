package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"log"
	"net"
	"sync"

	"github.com/tg123/phabrik/serialization"
)

type MessageHandler func(*Client, *ByteArrayMessage)

type Config struct {
	MessageHandlers map[MessageActorType]MessageHandler
	TLS             *tls.Config
	FrameHeaderCRC  bool
	FrameBodyCRC    bool
}

type Client struct {
	handlerLock     sync.Mutex
	messageHandlers map[MessageActorType]MessageHandler
	conn            net.Conn
	requestTable    sync.Map
	msgfac          *messageFactory

	frameRCfg frameReadConfig
	frameWCfg frameWriteConfig
}

func Dial(addr string, config Config) (*Client, error) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return Connect(conn, config)
}

func Connect(conn net.Conn, config Config) (*Client, error) {
	mf, err := newMessageFactory()
	if err != nil {
		return nil, err
	}

	c := &Client{
		msgfac: mf,
		conn:   conn,
	}

	if config.MessageHandlers != nil {
		c.messageHandlers = config.MessageHandlers
	} else {
		c.messageHandlers = make(map[MessageActorType]MessageHandler)
	}

	c.frameWCfg.SecurityProviderMask = securityProviderNone
	c.frameRCfg.CheckFrameHeaderCRC = true
	c.frameWCfg.FrameHeaderCRC = true

	if config.TLS != nil {
		tlsconn, err := createTlsConn(conn, mf, config.TLS)
		if err != nil {
			return nil, err
		}

		// secure conn does not do crc
		c.frameRCfg.CheckFrameHeaderCRC = false
		c.frameRCfg.CheckFrameBodyCRC = false
		c.frameWCfg.FrameHeaderCRC = false
		c.frameWCfg.FrameBodyCRC = false
		c.frameWCfg.SecurityProviderMask = securityProviderSsl

		c.conn = tlsconn
	}

	return c, nil
}

func (c *Client) SetMessageHandler(actor MessageActorType, h MessageHandler) {
	c.handlerLock.Lock()
	defer c.handlerLock.Unlock()

	if h == nil {
		delete(c.messageHandlers, actor)
	} else {
		c.messageHandlers[actor] = h
	}
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) handleTransportMessage(msg *ByteArrayMessage) error {
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

			err := c.SendOneWay(context.Background(), msg)
			if err != nil {
				return err
			}
		}
	default:
	}
	return nil
}

func (c *Client) Run(ctx context.Context) error {

	// transport init msg
	{
		msg := c.msgfac.newMessage()
		msg.Headers.Actor = MessageActorTypeTransport
		msg.Headers.HighPriority = true
		msg.Body = &struct {
			Address                string
			Instance               uint64
			Nonce                  serialization.GUID
			HeartbeatSupported     bool
			ConnectionFeatureFlags uint32
		}{
			HeartbeatSupported:     true,
			ConnectionFeatureFlags: 1,
		}
		err := c.SendOneWay(ctx, msg)
		if err != nil {
			return err
		}
	}

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
			ch, ok := c.requestTable.Load(headers.RelatesTo.String())

			if ok {
				ch.(chan *ByteArrayMessage) <- msg
			} else {
				log.Printf("unknown reply %v", headers.RelatesTo)
			}
		} else {
			c.handlerLock.Lock()
			if h, ok := c.messageHandlers[headers.Actor]; ok {
				go h(c, msg)
			}
			c.handlerLock.Unlock()
		}
	}
}

func (c *Client) SendOneWay(ctx context.Context, message *Message) error {
	c.msgfac.fillMessageId(message)
	return writeMessageWithFrame(c.conn, message, c.frameWCfg)
}

func (c *Client) RequestReply(ctx context.Context, message *Message) (*ByteArrayMessage, error) {
	c.msgfac.fillMessageId(message)
	message.Headers.ExpectsReply = true
	id := message.Headers.Id.String()
	defer c.requestTable.Delete(id)

	ch := make(chan *ByteArrayMessage)
	c.requestTable.Store(id, ch)

	err := c.SendOneWay(ctx, message)

	if err != nil {
		return nil, err
	}

	return <-ch, nil
}
