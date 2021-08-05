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
	Instance             NodeInstance // The node instance of the node which this information is of.
	Phase                NodePhase    // The phase of the node.
	Address              string       // The address of the node.
	Token                RoutingToken // The token owned by this node.
	LeaseAgentAddress    string       // The address of the lease agent for this node.
	LeaseAgentInstanceId int64        // Instance of the lease agent.
	EndToEnd             bool
	NodeFaultDomainId    common.Uri // The fault domain setting of this node.
	RingName             string
	Flags                int32
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
