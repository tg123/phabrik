package transport

import (
	"log"
	"net"

	"github.com/tg123/phabrik/serialization"
)

type Server struct {
	listener        net.Listener
	messageCallback MessageCallback
}

func ListenTCP(laddr string, config Config) (*Server, error) {
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		return nil, err
	}

	return Listen(l, config)
}

func Listen(l net.Listener, config Config) (*Server, error) {
	if config.TLS != nil {
		panic("not impl")
	}

	return &Server{
		listener:        l,
		messageCallback: config.MessageCallback,
	}, nil
}

func (s *Server) onMessage(conn Conn, msg *ByteArrayMessage) {
	if s.messageCallback != nil {
		go s.messageCallback(conn, msg)
	}
}

func (s *Server) handle(conn net.Conn) error {
	defer conn.Close()

	c, err := newConnection()
	if err != nil {
		return err
	}

	c.messageCallback = s.onMessage
	c.conn = conn

	nonce, err := serialization.NewGuidV4()
	if err != nil {
		return err
	}

	if err := c.sendTransportInit(&transportInitMessageBody{
		Address:                conn.LocalAddr().String(),
		Nonce:                  nonce,
		HeartbeatSupported:     true,
		ConnectionFeatureFlags: 1,
	}); err != nil {
		return err
	}

	return c.Wait()
}

func (s *Server) Serve() error {

	for {
		c, err := s.listener.Accept()
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Temporary() {
				log.Printf("accepting error %v", err)
				continue
			}

			return err
		}

		go s.handle(c)
	}
}

func (s *Server) Close() error {
	return s.listener.Close()
}
