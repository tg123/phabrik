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

type MessageHandler func(MessageActorType, *Message) error

type Config struct {
	MessageHandler MessageHandler
	TLS            *tls.Config
	FrameHeaderCRC bool
	FrameBodyCRC   bool
}

type Client struct {
	MessageHandler MessageHandler
	conn           net.Conn
	requestTable   sync.Map
	msgfac         *messageFactory

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
		msgfac:         mf,
		MessageHandler: config.MessageHandler,
		conn:           conn,
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

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) handleTransportMessage(msg *Message) error {
	body, ok := msg.Body.([]byte)
	if !ok {
		return fmt.Errorf("body not []byte")
	}

	switch msg.Headers.Action {
	case "HeartbeatRequest":
		var b struct {
			HeartbeatTimeTick int64
		}

		if err := serialization.Unmarshal(body, &b); err != nil {
			return err
		}

		{
			msg := c.msgfac.newMessage()
			msg.Headers.Actor = MessageActorTypeTransport
			msg.Headers.HighPriority = true
			msg.Headers.Action = "HeartbeatResponse"
			msg.Body = &b

			err := c.SendOneWay(context.TODO(), msg)
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
		tcpheader, tcpbody, err := nextFrame(c.conn, c.frameRCfg)
		if err != nil {
			return err
		}

		headers, err := parseFabricMessageHeaders(bytes.NewBuffer(tcpbody[:tcpheader.HeaderLength]))
		if err != nil {
			return err
		}

		body := tcpbody[tcpheader.HeaderLength:]

		msg := &Message{
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
				ch.(chan *Message) <- msg
			} else {
				log.Printf("unknown reply %v", headers.RelatesTo)
			}
		} else {
			if c.MessageHandler != nil {
				go func() {
					err := c.MessageHandler(headers.Actor, msg)
					if err != nil {
						log.Printf("handler err %v", err)
					}
				}()
			}
		}
	}
}

func (c *Client) SendOneWay(ctx context.Context, message *Message) error {
	c.msgfac.fillMessageId(message)
	return writeMessageWithFrame(c.conn, message, c.frameWCfg)
}

func (c *Client) RequestReply(ctx context.Context, message *Message) (*Message, error) {
	c.msgfac.fillMessageId(message)
	message.Headers.ExpectsReply = true
	id := message.Headers.Id.String()
	defer c.requestTable.Delete(id)

	ch := make(chan *Message)
	c.requestTable.Store(id, ch)

	err := c.SendOneWay(ctx, message)

	if err != nil {
		return nil, err
	}

	return <-ch, nil
}
