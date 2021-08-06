package federation

import "github.com/tg123/phabrik/common"

type PartnerNodeInfo struct {
	Instance             NodeInstance // The node instance of the node which this information is of.
	Phase                NodePhase    // The phase of the node.
	Address              string       // The address of the node.
	Token                RoutingToken // The token owned by this node.
	LeaseAgentAddress    string       // The address of the lease agent for this node.
	LeaseAgentInstanceId int64        // Instance of the lease agent.
	EndToEnd             bool
	NodeFaultDomainId    common.Uri // The fault domain setting of this node.
	RingName             string
}

func (s *SiteNode) updatePartnerNode(h *FederationPartnerNodeHeader) {
	s.partenersRWLock.RLock()
	old, found := s.parteners[h.Instance.Id]
	s.partenersRWLock.RUnlock()

	if !found || old.Instance.InstanceId < h.Instance.InstanceId {
		p := h.PartnerNodeInfo
		s.partenersRWLock.Lock()
		s.parteners[h.Instance.Id] = &p
		s.partenersRWLock.Unlock()
		s.onPartnerChanged(p, !found)
	}
}

func (s *SiteNode) knownPartnerNodes(filter func(*PartnerNodeInfo) bool) []*PartnerNodeInfo {
	s.partenersRWLock.RLock()
	defer s.partenersRWLock.RUnlock()
	var lst []*PartnerNodeInfo

	for _, p := range s.parteners {
		if filter(p) {
			lst = append(lst, p)
		}
	}

	return lst
}

func (s *SiteNode) closestPartnerNode(id NodeID, filter func(*PartnerNodeInfo) bool) *PartnerNodeInfo {
	// TODO optimize
	var cur *PartnerNodeInfo
	max := u128max

	for _, p := range s.knownPartnerNodes(filter) {
		diff := uint128(p.Instance.Id).sub(uint128(id))
		if diff == u128zero {
			return p
		}

		if diff.cmp(max) < 0 {
			max = diff
			cur = p
		}
	}

	return cur
}
