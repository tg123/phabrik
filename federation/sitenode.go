package federation

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/tg123/phabrik/lease"
	"github.com/tg123/phabrik/transport"
)

type SiteNode struct {
	instance  NodeInstance
	seedNodes []SeedNodeInfo

	transportServer *transport.Server
	leaseAgent      *lease.Agent

	phase        NodePhase
	phaseLock    sync.RWMutex
	phaseChanged chan int

	messageIdGenerator transport.MessageIdGenerator
	requestTable       transport.RequestTable
	clientDialer       Dialer
	connPool           sync.Map

	parteners       map[NodeID]*PartnerNodeInfo
	partenersRWLock sync.RWMutex
}

type SeedNodeInfo struct {
	Id      NodeID
	Address string
}

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

	msgfac, err := transport.NewMessageIdGenerator()
	if err != nil {
		return nil, err
	}

	s := SiteNode{
		seedNodes:          make([]SeedNodeInfo, len(config.SeedNodes)),
		instance:           config.Instance,
		transportServer:    config.TransportServer,
		leaseAgent:         config.LeaseAgent,
		phase:              NodePhaseBooting,
		phaseChanged:       make(chan int),
		parteners:          make(map[NodeID]*PartnerNodeInfo),
		messageIdGenerator: msgfac,
	}

	copy(s.seedNodes, config.SeedNodes)

	s.transportServer.SetMessageCallback(s.onMessage)
	// s.routing.onPartnerChanged = s.onPartnerChanged

	s.clientDialer = config.ClientDialer
	if s.clientDialer == nil {
		s.clientDialer = func(addr string) (*transport.Client, error) {
			return transport.DialTCP(addr, transport.ClientConfig{
				Config: transport.Config{
					TLS: config.ClientTLS,
				},
			})
		}
	}

	for _, seed := range config.SeedNodes {
		s.updatePartnerNode(&FederationPartnerNodeHeader{
			PartnerNodeInfo: PartnerNodeInfo{
				Instance: NodeInstance{
					Id: seed.Id,
				},
				Address: seed.Address,
			},
		})
	}

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

func (s *SiteNode) onPartnerChanged(p PartnerNodeInfo, isNew bool) {
	// found at least 1 R node, start join
	if p.Phase == NodePhaseRouting && s.phase == NodePhaseBooting {
		s.changePhase(NodePhaseJoining)
	}
}

func (s *SiteNode) onMessage(conn transport.Conn, bam *transport.ByteArrayMessage) {
	// if bam.Headers.Actor != transport.MessageActorTypeFederation {
	// 	return
	// }

	// fmt.Println("onmessage", bam.Headers)

	partners := bam.Headers.GetCustomHeaders(transport.MessageHeaderIdTypeFederationPartnerNode)
	for _, p := range partners {
		if p, ok := p.(*FederationPartnerNodeHeader); ok {
			s.updatePartnerNode(p)
		}
	}

	if s.requestTable.Feed(bam) {
		return
	}

	// TODO other handler

}

func (s *SiteNode) Serve() error {
	ch := make(chan error, 2)

	go func() {
		ch <- s.transportServer.Serve()
	}()

	go func() {
		ch <- s.leaseAgent.Wait()
	}()

	return <-ch
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
