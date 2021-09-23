package transport

import (
	"bytes"
	"fmt"
	"log"
	"net"
)

type PiperConfig struct {
	Config
	Filter       MessageTransformer
	FindUpstream func(initheaders *MessageHeaders, conn net.Conn) (net.Conn, Config, error)
}

type Piper struct {
	listener net.Listener
	config   PiperConfig
}

type MessageTransformer func(src, dst net.Conn, msg *ByteArrayMessage) *Message

func NewPiper(l net.Listener, config PiperConfig) (*Piper, error) {

	if config.FindUpstream == nil {
		return nil, fmt.Errorf("FindUpstream must not be nil")
	}

	return &Piper{
		listener: l,
		config:   config,
	}, nil
}

func (p *Piper) handle(conn net.Conn) error {
	defer conn.Close()

	headers, body, err := nextMessageHeaderAndBodyFromFrame(conn, frameReadConfig{
		CheckFrameHeaderCRC: true,
		CheckFrameBodyCRC:   false,
	})

	if err != nil {
		return nil
	}

	d, err := tapAcceptedConn(conn, p.config.Config, body)
	if err != nil {
		return err
	}

	rawu, uc, err := p.config.FindUpstream(headers, d.conn)
	if err != nil {
		return err
	}

	u, err := tapClientConn(rawu, uc)
	if err != nil {
		return err
	}

	if p.config.TLS == nil {
		// pass first message is unused if not secure conn
		if err := u.writeMessageWithFrame(&Message{
			Headers: *headers,
			Body:    body,
		}); err != nil {
			return nil
		}
	}

	return p.pipe(u, d)
}

func (p *Piper) Serve() error {
	for {
		c, err := p.listener.Accept()
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Temporary() {
				log.Printf("accepting error %v", err)
				continue
			}

			return err
		}

		go func() {
			if err := p.handle(c); err != nil {
				log.Printf("pipe err %v", err)
			}
		}()
	}
}

func (p *Piper) pipe(src, dst *connection) error {
	ch := make(chan error, 2)
	defer src.Close()
	defer dst.Close()

	go func() {
		ch <- p.copy(src, dst)
	}()

	go func() {
		ch <- p.copy(dst, src)
	}()

	return <-ch
}

func (p *Piper) copy(src, dst *connection) error {
	for {
		frameheader, framebody, err := nextFrame(src.conn, src.frameRCfg)
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

		var newmsg *Message

		if p.config.Filter != nil {
			newmsg = p.config.Filter(src.conn, dst.conn, msg)

			if newmsg == nil {
				// drop message
				continue
			}
		}

		if newmsg == nil {
			newmsg = &Message{
				Headers: msg.Headers,
				Body:    msg.Body,
			}
		}

		if err := writeMessageWithFrame(dst.conn, newmsg, dst.frameWCfg); err != nil {
			return err
		}
	}
}
