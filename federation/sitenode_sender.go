package federation

import (
	"context"
	"fmt"
	"sync"

	"github.com/tg123/phabrik/common"
	"github.com/tg123/phabrik/transport"
)

type Dialer func(addr string) (*transport.Client, error)

type cachedConn struct {
	parent  *SiteNode
	partner *PartnerNodeInfo
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
		conn, err := c.parent.clientDialer(c.partner.Address)

		if err != nil {
			c.lasterr = err
			c.Close()
			return
		}

		conn.SetMessageCallback(c.parent.onMessage)

		go func() {
			defer c.Close()
			conn.Wait()
		}()

		c.conn = conn
	})

	return c.conn // maybe nil and will recreate if err
}

func (c *cachedConn) Close() error {
	cc, ok := c.parent.connPool.LoadAndDelete(c.partner.Address)
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

func (s *SiteNode) getConn(p *PartnerNodeInfo) (*transport.Client, error) {
	c, _ := s.connPool.LoadOrStore(p.Address, &cachedConn{
		parent:  s,
		partner: p,
	})

	cc := c.(*cachedConn)
	return cc.Conn(), cc.lasterr
}

func (s *SiteNode) connectToNode(id NodeID, match bool) (*transport.Client, *PartnerNodeInfo, error) {

	t := s.closestPartnerNode(id, func(p *PartnerNodeInfo) bool {
		if p.Instance.Id == s.instance.Id {
			return false
		}

		return true
	})

	if (t == nil) || (match && t.Instance.Id != id) {
		return nil, nil, fmt.Errorf("%v not found", id)
	}

	c, err := s.getConn(t)
	return c, t, err
}

func (s *SiteNode) appendPartnerInfo(msg *transport.Message) {

	// add self
	msg.Headers.AppendCustomHeader(transport.MessageHeaderIdTypeFederationPartnerNode, &FederationPartnerNodeHeader{
		PartnerNodeInfo: PartnerNodeInfo{
			Instance:          s.instance,
			Address:           s.transportServer.Addr().String(),
			LeaseAgentAddress: s.leaseAgent.Addr().String(),
			Phase:             s.phase,
		},
	})

	for _, p := range s.knownPartnerNodes(func(pn *PartnerNodeInfo) bool {
		return pn.Instance.InstanceId > 0
	}) {
		msg.Headers.AppendCustomHeader(transport.MessageHeaderIdTypeFederationPartnerNode, &FederationPartnerNodeHeader{
			PartnerNodeInfo: *p,
		})
	}
}

func (s *SiteNode) fillMessageId(message *transport.Message) {
	if message.Headers.Id.IsEmpty() {
		message.Headers.Id = s.messageIdGenerator.Next()
	}
}

func (s *SiteNode) SendOneWay(id NodeID, msg *transport.Message) error {
	c, t, err := s.connectToNode(id, true)
	if err != nil {
		return err
	}

	s.fillMessageId(msg)
	s.appendPartnerInfo(msg)

	msg.Headers.SetCustomHeader(transport.MessageHeaderIdTypePToP, &PToPHeader{
		From:          s.instance,
		To:            t.Instance,
		Actor:         PToPActorDirect,
		ExactInstance: true,
	})

	return c.SendOneWay(msg)
}

func (s *SiteNode) Route(ctx context.Context, id NodeID, msg *transport.Message) (*transport.ByteArrayMessage, error) {
	c, t, err := s.connectToNode(id, false)
	if err != nil {
		return nil, err
	}

	s.fillMessageId(msg)
	s.appendPartnerInfo(msg)

	msg.Headers.SetCustomHeader(transport.MessageHeaderIdTypePToP, &PToPHeader{
		From:          s.instance,
		To:            t.Instance,
		Actor:         PToPActorRouting,
		ExactInstance: false,
	})

	msg.Headers.SetCustomHeader(transport.MessageHeaderIdTypeRouting, &RoutingHeader{
		From:            s.instance,
		To:              t.Instance,
		MessageId:       msg.Headers.Id,
		UseExactRouting: false,
		ExpectsReply:    true,
		Expiration:      common.TimeSpanMax,
		RetryTimeout:    common.TimeSpanMax,
	})

	msg.Headers.ExpectsReply = true

	pr := s.requestTable.Put(msg)
	defer pr.Close()

	if err := c.SendOneWay(msg); err != nil {
		return nil, err
	}

	reply, err := pr.Wait(ctx)
	if err != nil {
		return nil, err
	}

	if reply.Headers.ErrorCode != transport.FabricErrorCodeSuccess {
		return reply, fmt.Errorf("reply message contains err header, code: %v", reply.Headers.ErrorCode)
	}

	return reply, nil
}
