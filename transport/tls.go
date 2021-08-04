package transport

import (
	"bytes"
	"crypto/tls"
	"net"
	"time"
)

type fabricSecureConn struct {
	rawconn            net.Conn
	handshakeCompleted bool
	negoSend           bool
	rbuf               bytes.Buffer
	mf                 *messageFactory
	frameRCfg          frameReadConfig
	frameWCfg          frameWriteConfig
}

func createTlsConn(conn net.Conn, mf *messageFactory, tlsconf *tls.Config, factory func(conn net.Conn, config *tls.Config) *tls.Conn) (*tls.Conn, error) {
	rawtls := &fabricSecureConn{
		rawconn: conn,
		mf:      mf,
	}
	rawtls.frameWCfg.SecurityProviderMask = securityProviderSsl
	tlsconn := factory(rawtls, tlsconf)

	if err := tlsconn.Handshake(); err != nil {
		return nil, err
	}

	rawtls.markHandshakeComplete()

	return tlsconn, nil
}

func createTlsClientConn(conn net.Conn, mf *messageFactory, tlsconf *tls.Config) (*tls.Conn, error) {
	return createTlsConn(conn, mf, tlsconf, tls.Client)
}

func createTlsServerConn(conn net.Conn, mf *messageFactory, tlsconf *tls.Config) (*tls.Conn, error) {
	return createTlsConn(conn, mf, tlsconf, tls.Server)
}

func (c *fabricSecureConn) handshakeComplete() bool {
	return c.handshakeCompleted
}

func (c *fabricSecureConn) markHandshakeComplete() {
	c.handshakeCompleted = true
}

func (c *fabricSecureConn) Read(b []byte) (n int, err error) {
	if c.rbuf.Len() > 0 {
		return c.rbuf.Read(b)
	} else if !c.handshakeComplete() {
		tcpheader, tcpbody, err := nextFrame(c.rawconn, c.frameRCfg)
		if err != nil {
			return 0, err
		}

		if _, err := c.rbuf.Write(tcpbody[tcpheader.HeaderLength:]); err != nil {
			return 0, err
		}

		return c.rbuf.Read(b)
	} else {
		return c.rawconn.Read(b)
	}
}

func (c *fabricSecureConn) Write(b []byte) (n int, err error) {
	if !c.handshakeComplete() {
		msg := c.mf.newMessage()
		msg.Headers.Actor = MessageActorTypeSecurityContext
		msg.Body = b

		if !c.negoSend {
			msg.Headers.SetCustomHeader(MessageHeaderIdTypeSecurityNegotiation, &securityNegotiationHeader{
				X509ExtraFramingEnabled:  true,
				FramingProtectionEnabled: true, // here must be true to work on both windows and linux
			})
		}

		if writeMessageWithFrame(c.rawconn, msg, c.frameWCfg); err != nil {
			return 0, err
		}

		c.negoSend = true

		return len(b), nil
	} else {
		return c.rawconn.Write(b)
	}
}

func (c *fabricSecureConn) Close() error {
	return c.rawconn.Close()
}

func (c *fabricSecureConn) LocalAddr() net.Addr {
	return c.rawconn.LocalAddr()
}

func (c *fabricSecureConn) RemoteAddr() net.Addr {
	return c.rawconn.RemoteAddr()
}

func (c *fabricSecureConn) SetDeadline(t time.Time) error {
	return c.rawconn.SetDeadline(t)
}

func (c *fabricSecureConn) SetReadDeadline(t time.Time) error {
	return c.rawconn.SetReadDeadline(t)
}

func (c *fabricSecureConn) SetWriteDeadline(t time.Time) error {
	return c.rawconn.SetWriteDeadline(t)
}
