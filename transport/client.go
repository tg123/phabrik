package transport

import (
	"net"
)

type Client struct {
	Conn
}

func DialTCP(addr string, config Config) (*Client, error) {
	conn, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}

	return Connect(conn, config)
}

func Connect(conn net.Conn, config Config) (*Client, error) {

	c, err := newConnection()
	if err != nil {
		return nil, err
	}

	c.messageCallback = config.MessageCallback

	if config.TLS != nil {
		tlsconn, err := createTlsConn(conn, c.msgfac, config.TLS)
		if err != nil {
			return nil, err
		}

		c.setTls()
		c.conn = tlsconn
	} else {
		c.conn = conn
	}

	if err := c.sendTransportInit(&transportInitMessageBody{
		HeartbeatSupported:     true,
		ConnectionFeatureFlags: 1,
	}); err != nil {
		return nil, err
	}

	return &Client{
		Conn: c,
	}, nil
}
