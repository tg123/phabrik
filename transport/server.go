package transport

import (
	"log"
	"net"
)

type Server struct {
	listener        net.Listener
	messageCallback MessageCallback
	config          ServerConfig
}

type ServerConfig struct {
	Config
	MessageCallback MessageCallback
}

func ListenTCP(addr string, config ServerConfig) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return Listen(l, config)
}

func Listen(l net.Listener, config ServerConfig) (*Server, error) {
	return &Server{
		listener:        l,
		messageCallback: config.MessageCallback,
		config:          config,
	}, nil
}

func (s *Server) Addr() net.Addr {
	return s.listener.Addr()
}

func (s *Server) onMessage(conn Conn, msg *ByteArrayMessage) {
	if s.messageCallback != nil {
		go s.messageCallback(conn, msg)
	}
}

func (s *Server) handle(conn net.Conn) error {
	defer conn.Close()

	c, err := tapAcceptedConn(conn, s.config.Config, nil)
	if err != nil {
		return err
	}

	c.messageCallback = s.onMessage

	return c.Wait()
}

func (s *Server) SetMessageCallback(cb MessageCallback) {
	s.messageCallback = cb
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
