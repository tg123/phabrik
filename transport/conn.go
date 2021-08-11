package transport

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/tg123/phabrik/serialization"
)

type MessageCallback func(Conn, *ByteArrayMessage)

type Config struct {
	MessageCallback MessageCallback
	TLS             *tls.Config
	FrameHeaderCRC  bool
	FrameBodyCRC    bool
}

type Conn interface {
	SetMessageCallback(cb MessageCallback)

	SendOneWay(message *Message) error

	RequestReply(ctx context.Context, message *Message) (*ByteArrayMessage, error)

	// RequestReplyWithTable(ctx context.Context, message *Message, table *RequestTable) (*ByteArrayMessage, error)

	Ping(ctx context.Context) (time.Duration, error)

	Wait() error

	Close() error
}

type connection struct {
	messageCallback MessageCallback
	conn            net.Conn
	requestTable    RequestTable
	pinglock        sync.Mutex
	pingCh          chan int64
	msgfac          *messageFactory

	frameRCfg frameReadConfig
	frameWCfg frameWriteConfig

	closeOnce sync.Once
	fatalerr  error
}

func newConnection() (*connection, error) {
	mf, err := newMessageFactory()
	if err != nil {
		return nil, err
	}

	c := &connection{
		msgfac: mf,
		pingCh: make(chan int64),
	}

	c.frameWCfg.SecurityProviderMask = securityProviderNone
	c.frameRCfg.CheckFrameHeaderCRC = true
	c.frameWCfg.FrameHeaderCRC = true

	return c, nil
}

func (c *connection) setTls() {
	c.frameRCfg.CheckFrameHeaderCRC = false
	c.frameRCfg.CheckFrameBodyCRC = false
	c.frameWCfg.FrameHeaderCRC = false
	c.frameWCfg.FrameBodyCRC = false
	c.frameWCfg.SecurityProviderMask = securityProviderSsl
}

func (c *connection) SetMessageCallback(cb MessageCallback) {
	c.messageCallback = cb
}

func (c *connection) Close() error {
	err := c.conn.Close()

	c.closeOnce.Do(func() {
		close(c.pingCh)

		c.requestTable.Close()

		// c.requestTable.Range(func(key, value interface{}) bool {
		// 	if ch, ok := value.(chan *ByteArrayMessage); ok {
		// 		close(ch)
		// 	}

		// 	return true
		// })
	})

	return err
}

type heartbeat struct {
	HeartbeatTimeTick int64
}

func (c *connection) Ping(ctx context.Context) (time.Duration, error) {
	c.pinglock.Lock()
	defer c.pinglock.Unlock()

	var b heartbeat
	b.HeartbeatTimeTick = time.Now().UnixNano()

	msg := c.msgfac.newMessage()
	msg.Headers.Actor = MessageActorTypeTransport
	msg.Headers.HighPriority = true
	msg.Headers.Action = "HeartbeatRequest"
	msg.Body = &b

	err := c.SendOneWay(msg)
	if err != nil {
		return -1, err
	}

	select {
	case <-ctx.Done():
		return -1, ctx.Err()
	case t := <-c.pingCh:
		if t != b.HeartbeatTimeTick {
			return -1, fmt.Errorf("heartbeak time tick out of order")
		}
		return time.Since(time.Unix(0, t)), nil
	}
}

func (c *connection) handleTransportMessage(msg *ByteArrayMessage) error {
	switch msg.Headers.Action {
	case "HeartbeatRequest":

		resp := c.msgfac.newMessage()
		resp.Headers.Actor = MessageActorTypeTransport
		resp.Headers.HighPriority = true
		resp.Headers.Action = "HeartbeatResponse"
		resp.Body = msg.Body

		err := c.SendOneWay(resp)
		if err != nil {
			return err
		}
	case "HeartbeatResponse":
		var b heartbeat

		if err := serialization.Unmarshal(msg.Body, &b); err != nil {
			return err
		}

		c.pingCh <- b.HeartbeatTimeTick
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

		if headers.Actor == MessageActorTypeTransport {
			go c.handleTransportMessage(msg)
			continue
		}

		// TODO support server side reject
		if headers.Actor == MessageActorTypeTransportSendTarget && headers.Action == "ConnectionAuth" {
			if headers.ErrorCode != FabricErrorCodeSuccess {
				var b struct {
					Message string
				}

				serialization.Unmarshal(body, &b) // ignore error
				c.fatalerr = fmt.Errorf("connection auth failure, error code [%v], msg [%v]", headers.ErrorCode, b.Message)

				return c.Close()
			}
		}

		if !c.requestTable.Feed(msg) {
			if c.messageCallback != nil {
				c.messageCallback(c, msg)
			}
		}

		// if !headers.RelatesTo.IsEmpty() {
		// 	ch, ok := c.requestTable.LoadAndDelete(headers.RelatesTo.String())

		// 	if ok {
		// 		ch.(chan *ByteArrayMessage) <- msg
		// 	} else {
		// 		log.Printf("unknown reply %v", headers.RelatesTo)
		// 	}
		// } else {
		// }
	}
}

func (c *connection) SendOneWay(message *Message) error {
	if c.fatalerr != nil {
		return c.fatalerr
	}
	c.msgfac.fillMessageId(message)
	return writeMessageWithFrame(c.conn, message, c.frameWCfg)
}

func (c *connection) RequestReply(ctx context.Context, message *Message) (*ByteArrayMessage, error) {
	c.msgfac.fillMessageId(message)
	message.Headers.ExpectsReply = true
	pr := c.requestTable.Put(message)
	defer pr.Close()

	if err := c.SendOneWay(message); err != nil {
		return nil, err
	}

	return pr.Wait(ctx)
}

// func (c *connection) RequestReplyWithTable(ctx context.Context, message *Message, table *RequestTable) (*ByteArrayMessage, error) {
// 	if c.fatalerr != nil {
// 		return nil, c.fatalerr
// 	}
// 	c.msgfac.fillMessageId(message)

// 	// id := message.Headers.Id.String()
// 	// defer c.requestTable.Delete(id)

// 	// ch := make(chan *ByteArrayMessage)
// 	// c.requestTable.Store(id, ch)

// 	pr := table.Put(message)
// 	defer pr.Close()

// 	err := writeMessageWithFrame(c.conn, message, c.frameWCfg)

// 	if err != nil {
// 		return nil, err
// 	}

// 	return pr.Wait(ctx)

// 	// select {
// 	// case <-ctx.Done():
// 	// 	return nil, ctx.Err()
// 	// case reply := <-ch:
// 	// 	if reply == nil {
// 	// 		return nil, fmt.Errorf("operation cancelled")
// 	// 	}
// 	// 	return reply, nil
// 	// }

// }
