package federation

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tg123/phabrik/lease"
	"github.com/tg123/phabrik/transport"
)

type SiteNode struct {
	instance        NodeInstance
	phase           NodePhase
	phaseLock       sync.RWMutex
	phaseChanged    chan int
	routing         *routingTable
	transportServer *transport.Server
	leaseAgent      *lease.Agent
	// clientTLS       *tls.Config
	clientDialer Dialer
	connPool     sync.Map
	seedNodes    []SeedNodeInfo
}

type SeedNodeInfo struct {
	Id      NodeID
	Address string
}

type Dialer func(addr string) (*transport.Client, error)

type SiteNodeConfig struct {
	ClientTLS       *tls.Config
	ClientDialer    Dialer
	TransportServer *transport.Server
	LeaseAgent      *lease.Agent
	Instance        NodeInstance
	SeedNodes       []SeedNodeInfo
}

func NewSiteNode(config SiteNodeConfig) (*SiteNode, error) {

	if len(config.SeedNodes) == 0 {
		return nil, fmt.Errorf("empty seednodes")
	}

	s := SiteNode{
		instance:        config.Instance,
		transportServer: config.TransportServer,
		leaseAgent:      config.LeaseAgent,
		routing:         &routingTable{},
		phase:           NodePhaseBooting,
		phaseChanged:    make(chan int),
		seedNodes:       make([]SeedNodeInfo, len(config.SeedNodes)),
	}

	copy(s.seedNodes, config.SeedNodes)

	s.transportServer.SetMessageCallback(s.onMessage)
	s.routing.onPartnerChanged = s.onPartnerChanged

	s.clientDialer = config.ClientDialer
	if s.clientDialer == nil {
		s.clientDialer = func(addr string) (*transport.Client, error) {
			return transport.DialTCP(addr, transport.Config{
				TLS: config.ClientTLS,
			})
		}
	}

	for _, seed := range config.SeedNodes {
		s.routing.updatePartnerNode(&FederationPartnerNodeHeader{
			Instance: NodeInstance{
				Id: seed.Id,
			},
			Address: seed.Address,
		})
	}

	// add self
	s.routing.updatePartnerNode(&FederationPartnerNodeHeader{
		Instance:          s.instance,
		Address:           s.transportServer.Addr().String(),
		LeaseAgentAddress: s.leaseAgent.Addr().String(),
	})

	return &s, nil
}

func (s *SiteNode) changePhase(newPhase NodePhase) {
	if s.phase == newPhase {
		return
	}

	s.phaseLock.Lock()
	defer s.phaseLock.Unlock()

	s.phase = newPhase
	close(s.phaseChanged)
	s.phaseChanged = make(chan int)
}

func (s *SiteNode) onPartnerChanged(p partnerNodeInfo, isNew bool) {
	// found at least 1 R node, start join
	if p.Phase == NodePhaseRouting && s.phase == NodePhaseBooting {
		s.changePhase(NodePhaseJoining)
	}
}

func (s *SiteNode) onMessage(conn transport.Conn, bam *transport.ByteArrayMessage) {
	if bam.Headers.Actor != transport.MessageActorTypeFederation {
		return
	}

	partners := bam.Headers.GetCustomHeaders(transport.MessageHeaderIdTypeFederationPartnerNode)

	for _, p := range partners {
		if p, ok := p.(*FederationPartnerNodeHeader); ok {
			s.routing.updatePartnerNode(p)
		}
	}

}

func (s *SiteNode) Bootstrap(ctx context.Context) error {
	if s.phase >= NodePhaseJoining {
		return nil
	}

	for {
		for _, seed := range s.seedNodes {
			if err := s.votePing(seed.Id); err != nil {
				log.Printf("send vote ping to %v failed %v", seed.Address, err)
			}
		}

		select {
		case <-s.phaseChanged:
			if s.phase >= NodePhaseJoining {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * 30): // TODO config
			// continue
		}
	}

}

func (s *SiteNode) votePing(id NodeID) error {
	return s.SendOneWay(id, &transport.Message{
		Headers: transport.MessageHeaders{
			Actor:  transport.MessageActorTypeFederation,
			Action: "VotePing",
		},
	})
}

type cachedConn struct {
	parent  *SiteNode
	addr    string
	conn    *transport.Client
	init    sync.Once
	close   sync.Once
	lasterr error
}

func (c *cachedConn) Conn() *transport.Client {
	if c.conn != nil {
		return c.conn
	}

	c.init.Do(func() {
		conn, err := c.parent.clientDialer(c.addr)

		if err != nil {
			c.lasterr = err
			c.Close()
			return
		}

		go func() {
			defer c.Close()
			conn.Wait()
		}()

		c.conn = conn
	})

	return c.conn // maybe nil and will recreate if err
}

func (c *cachedConn) Close() error {
	cc, ok := c.parent.connPool.LoadAndDelete(c.addr)
	if !ok {
		return nil
	}

	cc.(*cachedConn).close.Do(func() {
		tc := cc.(*cachedConn).conn
		if tc != nil {
			tc.Close()
		}
	})

	return nil
}

func (s *SiteNode) getConn(addr string) (*transport.Client, error) {
	c, _ := s.connPool.LoadOrStore(addr, &cachedConn{
		parent: s,
		addr:   addr,
	})

	// log.Printf("cannot connect to %v, err: %v", c.addr, err)
	cc := c.(*cachedConn)
	return cc.Conn(), cc.lasterr
}

func (s *SiteNode) connectToNode(id NodeID, match bool) (*transport.Client, *partnerNodeInfo, error) {
	t := s.routing.closePartnerNode(id)
	if match && t.Instance.Id != id {
		return nil, nil, fmt.Errorf("%v not found", id)
	}

	c, err := s.getConn(t.Address)
	return c, &t, err
}

func (s *SiteNode) appendPartnerInfo(msg *transport.Message) {
	for _, p := range s.routing.knownPartnerNodes() {
		if p.Instance.InstanceId > 0 { // not dummy node
			ph := FederationPartnerNodeHeader(p)
			msg.Headers.AppendCustomHeader(transport.MessageHeaderIdTypeFederationPartnerNode, &ph)
		}
	}
}

func (s *SiteNode) SendOneWay(id NodeID, msg *transport.Message) error {
	c, t, err := s.connectToNode(id, true)
	if err != nil {
		return err
	}

	s.appendPartnerInfo(msg)

	msg.Headers.SetCustomHeader(transport.MessageHeaderIdTypePToP, &PToPHeader{
		From:          s.instance,
		To:            t.Instance,
		Actor:         PToPActorDirect,
		ExactInstance: true,
	})

	return c.SendOneWay(msg)
}

func (s *SiteNode) Serve() error {
	return s.transportServer.Serve()
}

func (s *SiteNode) Close() error {
	s.transportServer.Close()
	s.leaseAgent.Close()

	s.connPool.Range(func(key, value interface{}) bool {
		if c, ok := value.(*cachedConn); ok {
			c.Close()
		}

		return true
	})

	return nil
}
