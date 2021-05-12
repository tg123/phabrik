package transport

import (
	"net"
)

type Client struct {
	Conn
	MessageHandler
}

func Dial(network, addr string, config Config) (*Client, error) {
	conn, err := net.Dial(network, addr)

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

	c.initHandlers(config.MessageCallbacks)

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
		Conn:           c,
		MessageHandler: c,
	}, nil
}
