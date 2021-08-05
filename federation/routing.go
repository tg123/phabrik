package federation

import (
	"sort"
	"sync"
)

type partnerNodeInfo FederationPartnerNodeHeader

type routingTable struct {
	parteners        []partnerNodeInfo
	partenersRWLock  sync.RWMutex
	onPartnerChanged func(p partnerNodeInfo, isNew bool)
}

func (n NodeID) Less(v NodeID) bool {
	if v.Hi > n.Hi {
		return true
	} else if v.Hi == n.Hi {
		return v.Lo < n.Lo
	}

	return false
}

func (r *routingTable) updatePartnerNode(h *FederationPartnerNodeHeader) {

	hid := h.Instance.Id

	r.partenersRWLock.RLock()
	idx := sort.Search(len(r.parteners), func(i int) bool {
		return !r.parteners[i].Instance.Id.Less(hid)
	})
	r.partenersRWLock.RUnlock()

	if idx < len(r.parteners) && r.parteners[idx].Instance.Id == h.Instance.Id {
		// already in it update

		// current entry is older than incoming
		if r.parteners[idx].Instance.InstanceId < h.Instance.InstanceId {
			p := partnerNodeInfo(*h)
			r.partenersRWLock.Lock()
			r.parteners[idx] = p
			r.partenersRWLock.Unlock()
			r.onPartnerChanged(p, false)
		}
	} else {

		// not in list add
		r.partenersRWLock.Lock()

		p := partnerNodeInfo(*h)

		// https://stackoverflow.com/questions/46128016/insert-a-value-in-a-slice-at-a-given-index
		if len(r.parteners) == idx { // nil or empty slice or after last element
			r.parteners = append(r.parteners, p)
		} else {
			r.parteners = append(r.parteners[:idx+1], r.parteners[idx:]...)
			r.parteners[idx] = p
		}

		r.partenersRWLock.Unlock()
		r.onPartnerChanged(p, true)

	}
}

func (r *routingTable) knownPartnerNodes() []partnerNodeInfo {
	r.partenersRWLock.RLock()
	defer r.partenersRWLock.RUnlock()
	p := make([]partnerNodeInfo, len(r.parteners))
	copy(p, r.parteners)
	return p
}

func (r *routingTable) closePartnerNode(id NodeID) partnerNodeInfo {
	r.partenersRWLock.RLock()
	defer r.partenersRWLock.RUnlock()

	idx := sort.Search(len(r.parteners), func(i int) bool {
		return !r.parteners[i].Instance.Id.Less(id)
	})

	if idx >= len(r.parteners) {
		idx = 0 // first node own
	}

	return r.parteners[idx]
}
