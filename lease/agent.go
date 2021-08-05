package lease

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

func uniqId() int64 {
	// this id is a windows filetime, used in lease id
	return time.Now().UnixNano()/100 + 116444736000000000
}

type LeaseAgentState int32

const (
	LeaseAgentStateOpen LeaseAgentState = iota
	LeaseAgentStateSuspended
	LeaseAgentStateFailed
)

type AgentConfig struct {
	TLS                      *tls.Config
	AppId                    string
	LeaseDuration            time.Duration
	ApplicationLeaseDuration time.Duration
	LeaseSuspendTimeout      time.Duration
	ArbitrationTimeout       time.Duration
}

func (c *AgentConfig) SetDefault() {
	// TODO sync with federation.h
	c.LeaseDuration = 30 * time.Second
	c.ApplicationLeaseDuration = 30 * time.Second
	c.LeaseSuspendTimeout = 2 * time.Second
	c.ArbitrationTimeout = 30 * time.Second
}

type Dialer func(addr string) (net.Conn, error)

type Agent struct {
	LocalInstance int64
	config        AgentConfig
	marshaller    marshalContext
	sessions      sync.Map
	listener      net.Listener
	dial          Dialer
}

var errAgentClosed = fmt.Errorf("agent closed")

func closedDialer(addr string) (net.Conn, error) {
	return nil, errAgentClosed
}

func TcpDialer(addr string) (net.Conn, error) {
	return net.Dial("tcp", addr)
}

func NewTcpListeningAgent(config AgentConfig, listenAddr string) (*Agent, error) {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}

	return NewAgent(config, l, TcpDialer)
}

func NewAgent(config AgentConfig, listener net.Listener, dial Dialer) (*Agent, error) {
	a := Agent{
		dial:          dial,
		listener:      listener,
		LocalInstance: uniqId(),
	}
	a.config = config

	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		return nil, err
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}

	a.marshaller = marshalContext{
		AppId:   config.AppId,
		Address: host,
		Port:    uint16(p),
	}

	return &a, nil
}

func (a *Agent) Addr() net.Addr {
	return a.listener.Addr()
}

func (a *Agent) Establish(addr string) (Session, error) {
	s := leaseSession{
		addr:          addr,
		parent:        a,
		pingCh:        make(chan int),
		localInstance: uniqId(),
	}

	err := s.reconnect()
	if err != nil {
		return nil, err
	}

	a.sessions.Store(addr, &s)

	return &s, nil
}

func (a *Agent) Find(addr string) (Session, bool) {
	return nil, false
}

func (a *Agent) Wait() error {
	defer a.listener.Close()

	for {
		c, err := a.listener.Accept()

		if err != nil {
			continue
		}

		go a.handleIncoming(c)
	}
}

func (a *Agent) Close() error {
	a.dial = closedDialer
	a.listener.Close()
	a.sessions.Range(func(key, value interface{}) bool {
		if s, ok := value.(Session); ok {
			s.Close()
		}

		return true
	})

	return nil
}

func (a *Agent) handleIncoming(conn net.Conn) error {
	defer conn.Close()

	if a.config.TLS != nil {
		conn = tls.Server(conn, a.config.TLS)
	}

	if err := writeConnectFrame(conn); err != nil {
		return err
	}

	for {
		body, err := nextLtFrame(conn)
		if err != nil {
			return err
		}

		m, err := unmarshal(body)
		if err != nil {
			return err
		}

		s, ok := a.sessions.Load(m.MessageListenEndpoint)
		if !ok {
			continue
		}

		ls, ok := s.(*leaseSession)
		if ok {
			ls.onMessage(m)
		}
	}
}

type SessionMetadata struct {
	LocalInstance  int64
	RemoteInstance int64
	RemoteEndpoint string
}

type Session interface {
	Meta() SessionMetadata
	// State() LeaseAgentState
	Ping(ctx context.Context) error

	LastPongTime() time.Time

	Close() error
}

func PingLoop(ctx context.Context, sess Session, interval time.Duration) error {

	for {
		ctx0, _ := context.WithTimeout(ctx, interval)
		sess.Ping(ctx0) // TODO ignore error atm, introduce arbitrator later

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
			// continue
		}
	}
}

type leaseSession struct {
	addr           string
	parent         *Agent
	localInstance  int64
	remoteInstance int64

	closed    bool
	conn      net.Conn
	connected bool
	objLock   sync.Mutex

	pingCh   chan int
	pingLock sync.RWMutex
	lastpong time.Time
}

func (s *leaseSession) Meta() SessionMetadata {
	return SessionMetadata{}
}

// func (s *leaseSession) State() LeaseAgentState {
// 	return LeaseAgentStateFailed
// }

func (s *leaseSession) Close() error {
	s.parent.sessions.Delete(s.addr)
	s.objLock.Lock()
	defer s.objLock.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.conn.Close()
	s.boardcastPong() // unlock waiting pings

	return nil
}

func (s *leaseSession) reconnect() error {
	s.objLock.Lock()
	defer s.objLock.Unlock()

	if s.connected {
		return nil
	}

	if s.closed {
		return fmt.Errorf("session closed")
	}

	c, err := s.parent.dial(s.addr)
	if err != nil {
		return err
	}

	if s.parent.config.TLS != nil {
		c = tls.Client(c, s.parent.config.TLS)
	}

	s.conn = c
	s.connected = true
	return nil
}

func (s *leaseSession) Ping(ctx context.Context) error {
	msg := s.createPingMessage()

	data, err := s.parent.marshaller.marshal(msg)
	if err != nil {
		return err
	}

	if err := writeDataWithFrame(s.conn, data); err != nil {
		if neterr, ok := err.(net.Error); ok {
			if !neterr.Temporary() {
				s.objLock.Lock()
				s.connected = false
				s.objLock.Unlock()

				if err = s.reconnect(); err != nil {
					return err
				}

				return s.Ping(ctx)
			}
		}
		return err
	}

	s.pingLock.RLock()
	ch := s.pingCh
	s.pingLock.RUnlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}

func (s *leaseSession) LastPongTime() time.Time {
	return s.lastpong
}

func (s *leaseSession) onMessage(m *Message) {

	switch m.Type {
	case LeaseMessageTypePingRequest:
		// TODO
	case LeaseMessageTypePingResponse:
		s.lastpong = time.Now()
		s.remoteInstance = m.RemoteLeaseAgentInstance
		s.boardcastPong()
	default:
	}
}

func (s *leaseSession) boardcastPong() {
	s.pingLock.Lock()
	ch := s.pingCh
	s.pingCh = make(chan int)
	close(ch)
	s.pingLock.Unlock()
}

func (s *leaseSession) createPingMessage() *Message {
	message := Message{}
	message.Type = LeaseMessageTypePingRequest
	message.Expiration = s.parent.config.LeaseDuration // TODO confirm config entry
	message.LeaseInstance = s.localInstance
	message.RemoteLeaseAgentInstance = s.remoteInstance
	return &message
}
