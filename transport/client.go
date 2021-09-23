package transport

import (
	"net"
)

type Client struct {
	*connection
}

type ClientConfig struct {
	Config
	MessageCallback MessageCallback
}

func DialTCP(addr string, config ClientConfig) (*Client, error) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return Connect(conn, config)
}

func Connect(conn net.Conn, config ClientConfig) (*Client, error) {
	c, err := tapClientConn(conn, config.Config)
	if err != nil {
		return nil, err
	}

	c.messageCallback = config.MessageCallback

	return &Client{
		connection: c,
	}, nil
}
