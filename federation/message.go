package federation

import (
	"github.com/tg123/phabrik/common"
	"github.com/tg123/phabrik/transport"
)

func init() {
	transport.RegisterHeaderActivator(transport.MessageHeaderIdTypePToP, func() interface{} {
		return &PToPHeader{}
	})

	transport.RegisterHeaderActivator(transport.MessageHeaderIdTypeFederationPartnerNode, func() interface{} {
		return &FederationPartnerNodeHeader{}
	})

	transport.RegisterHeaderActivator(transport.MessageHeaderIdTypeRouting, func() interface{} {
		return &RoutingHeader{}
	})
}

type NodePhase int64

const (
	NodePhaseBooting NodePhase = iota
	NodePhaseJoining
	NodePhaseInserting
	NodePhaseRouting
	NodePhaseShutdown
)

type RoutingToken struct {
	Range   NodeIdRange
	Version uint64
}

type FederationPartnerNodeHeader struct {
	PartnerNodeInfo
	Flags int32
}

type BootingInfo struct {
	Leader NodeID
	Time   common.StopwatchTime
}

type PToPActor int64

const (
	PToPActorDirect PToPActor = iota
	PToPActorFederation
	PToPActorRouting
	PToPActorBroadcast
	PToPActorUpperBound
)

type PToPHeader struct {
	From          NodeInstance
	To            NodeInstance
	Actor         PToPActor
	FromRing      string
	ToRing        string
	ExactInstance bool
}

type RoutingHeader struct {
	From NodeInstance
	To   NodeInstance
	transport.MessageId
	UseExactRouting bool
	ExpectsReply    bool
	Expiration      common.TimeSpan
	RetryTimeout    common.TimeSpan
	FromRing        string
	ToRing          string
}

type TimeRange struct {
	Begin common.StopwatchTime
	End   common.StopwatchTime
}

type TicketGap struct {
	Range    NodeIdRange
	Interval TimeRange
}

type VoteTicket struct {
	VoteId     NodeID
	ExpireTime common.StopwatchTime
	Gaps       []TicketGap
}

type GlobalLease struct {
	Tickets  []VoteTicket
	Delta    common.TimeSpan
	BaseTime common.StopwatchTime
}
