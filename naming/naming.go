package naming

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/tg123/phabrik/common"
	"github.com/tg123/phabrik/federation"
	"github.com/tg123/phabrik/serialization"
	"github.com/tg123/phabrik/transport"
)

type ProtocolVersion struct {
	Major int64
	Minor int64
}

type ActivityId struct {
	Id    serialization.GUID
	Index uint64
}

func (a ActivityId) IsEmpty() bool {
	return a.Id.IsEmpty() && a.Index == 0
}

func (a ActivityId) String() string {
	return fmt.Sprintf("%v:%v", a.Id.String(), a.Index)
}

// func init() {
// 	transport.RegisterHeaderActivator(transport.MessageHeaderIdTypeFabricActivity, func() interface{} {
// 		return &ActivityId{}
// 	})
// }

func newNamingMessage(action string) (*transport.Message, error) {
	activityId, err := serialization.NewGuidV4()

	if err != nil {
		return nil, err
	}

	msg := &transport.Message{}
	msg.Headers.Actor = transport.MessageActorTypeNamingGateway
	msg.Headers.Action = action
	msg.Headers.ExpectsReply = true

	msg.Headers.SetCustomHeader(transport.MessageHeaderIdTypeFabricActivity, &struct {
		Activity ActivityId
	}{
		Activity: ActivityId{
			Id: activityId,
		},
	})

	msg.Headers.SetCustomHeader(transport.MessageHeaderIdTypeClientProtocolVersion, &ProtocolVersion{
		Major: 1,
		Minor: 2,
	})

	msg.Headers.SetCustomHeader(transport.MessageHeaderIdTypeTimeout, &struct {
		Timeout time.Duration
	}{
		Timeout: 20 * time.Millisecond,
	})

	return msg, nil
}

type NamingClient struct {
	OnServiceNotification func(notification *ServiceNotification)

	transport    *transport.Client
	nextFilterId uint64
	clientId     string
}

func NewNamingClient(transport *transport.Client) (*NamingClient, error) {
	guid, err := serialization.NewGuidV4()
	if err != nil {
		return nil, err
	}

	n := &NamingClient{
		transport:    transport,
		nextFilterId: 0,
		clientId:     "phabrik-" + guid.String(),
	}

	// TODO chain design
	transport.SetMessageCallback(n.onMessage)

	return n, nil
}

type GatewayDescription struct {
	Address      string
	NodeInstance federation.NodeInstance
	NodeName     string
}

type ServiceNotificationPageId struct {
	NotificationId transport.ActivityId
	PageIndex      uint64
}

type ConsistencyUnitId struct {
	GUID serialization.GUID
}

type ServiceTableEntry struct {
	ConsistencyUnitId ConsistencyUnitId
	ServiceName       string
	ServiceReplicaSet ServiceReplicaSet
	IsFound           bool
}

type ServiceReplicaSet struct {
	IsStateful             bool
	IsPrimaryLocationValid bool
	PrimaryLocation        string
	ReplicaLocations       []string
	LookupVersion          int64
	IsPrimaryAuxiliary     bool
	AuxiliaryLocations     []string
}

type ServiceTableEntryNotification struct {
	ServiceTable       *ServiceTableEntry
	MatchedPrimaryOnly bool
}

type ServiceNotification struct {
	NotificationPageId ServiceNotificationPageId
	Generation         GenerationNumber
	Versions           *VersionRangeCollection
	Partitions         []*ServiceTableEntryNotification
}

func (n *NamingClient) onMessage(conn transport.Conn, bam *transport.ByteArrayMessage) {
	switch bam.Headers.Action {
	case "ServiceNotificationRequest":
		var b struct {
			Notification *ServiceNotification
		}

		err := serialization.Unmarshal(bam.Body, &b)
		if err != nil {
			log.Printf("ServiceNotificationRequest unmarshal err %v", err)
		}

		reply, err := newNamingMessage("ServiceNotificationReply")
		if err != nil {
			log.Printf("ServiceNotificationRequest NewNamingMessage err %v", err)
		}

		reply.Headers.RelatesTo = bam.Headers.Id
		reply.Body = &struct {
			Error transport.FabricErrorCode
		}{
			transport.FabricErrorCodeSuccess,
		}

		conn.SendOneWay(reply)

		if n.OnServiceNotification != nil {
			go n.OnServiceNotification(b.Notification)
		}
	default:
		log.Printf("unsupported action %v", bam.Headers.Action)
	}
}

func (n *NamingClient) requestReply(ctx context.Context, msg *transport.Message) (*transport.ByteArrayMessage, error) {
	reply, err := n.transport.RequestReply(context.TODO(), msg)
	if err != nil {
		return nil, err
	}

	body := reply.Body
	if reply.Headers.Action == "ClientOperationFailure" {
		var b struct {
			ErrorCode int64
		}
		if err := serialization.Unmarshal(body, &b); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("ClientOperationFailure HResult: %v", b.ErrorCode)
	}

	return reply, nil
}

func (n *NamingClient) Ping(ctx context.Context) (*GatewayDescription, error) {
	msg, err := newNamingMessage("PingRequest")

	if err != nil {
		return nil, err
	}

	reply, err := n.requestReply(ctx, msg)
	if err != nil {
		return nil, err
	}

	body := reply.Body
	switch reply.Headers.Action {
	case "PingRequest":
		var b struct {
			GatewayDescription GatewayDescription
		}
		if err := serialization.Unmarshal(body, &b); err != nil {
			return nil, err
		}

		return &b.GatewayDescription, nil
	default:
		return nil, fmt.Errorf("unsupported action %v MessageActorTypeNamingGateway", reply.Headers.Action)
	}
}

type GenerationNumber struct {
	Generation int64
	Owner      federation.NodeID
}

type ServiceNotificationFilterFlags struct {
	Flags uint32
}

type ServiceNotificationFilter struct {
	FilterId uint64
	Name     common.Uri
	Flags    ServiceNotificationFilterFlags
}

type VersionRange struct {
	StartVersion int64
	EndVersion   int64
}

type VersionRangeCollection struct {
	VersionRanges []VersionRange
}

type NotificationClientConnectionRequestBody struct {
	ClientId       string
	Generation     GenerationNumber
	ClientVersions *VersionRangeCollection
	Filters        []*ServiceNotificationFilter
}

type NotificationClientConnectionReplyBody struct {
	Generation                       GenerationNumber
	LastDeletedEmptyPartitionVersion int64
	Gateway                          GatewayDescription
}

type RegisterServiceNotificationFilterRequestBody struct {
	ClientId string
	Filter   *ServiceNotificationFilter
}

func (n *NamingClient) RegisterFilter(ctx context.Context, name common.Uri, matchNamePrefix, matchPrimaryChangeOnly bool) (uint64, error) {

	filterId := atomic.AddUint64(&n.nextFilterId, 1)

	var flags uint32

	// TODO const
	if matchNamePrefix {
		flags |= 1
	}

	if matchPrimaryChangeOnly {
		flags |= 2
	}

	{
		msg, err := newNamingMessage("NotificationClientConnectionRequest")
		if err != nil {
			return 0, err
		}

		msg.Body = &NotificationClientConnectionRequestBody{
			ClientId:       n.clientId,
			ClientVersions: &VersionRangeCollection{},
		}

		_, err = n.requestReply(ctx, msg)
		if err != nil {
			return 0, err
		}
	}

	{
		msg, err := newNamingMessage("RegisterServiceNotificationFilterRequest")
		if err != nil {
			return 0, err
		}

		msg.Headers.SetCustomHeader(transport.MessageHeaderIdTypeClientIdentity, &struct {
			TargetName   string
			FriendlyName string
		}{
			TargetName:   "",
			FriendlyName: n.clientId,
		})

		msg.Body = &RegisterServiceNotificationFilterRequestBody{
			ClientId: n.clientId,
			Filter: &ServiceNotificationFilter{
				FilterId: filterId,
				Name:     name,
				Flags:    ServiceNotificationFilterFlags{flags},
			},
		}

		_, err = n.requestReply(ctx, msg)
		if err != nil {
			return 0, err
		}
	}

	return filterId, nil
}

type ApplicationQueryResult struct {
	ApplicationName        common.Uri
	ApplicationTypeName    string
	ApplicationTypeVersion string
	ApplicationStatus      int64
	HealthState            int64
	ApplicationParameters  map[string]string
}

func (n *NamingClient) GetApplicationList(ctx context.Context, filter string) ([]ApplicationQueryResult, error) {

	msg, err := newNamingMessage("QueryRequest")
	if err != nil {
		return nil, err
	}

	type QueryArgumentMapKey struct {
		Key string
	}

	param := &struct {
		QueryArgs struct {
			QueryArgumentMap map[QueryArgumentMapKey]string
		}
		QueryName string
	}{
		QueryName: "GetApplicationList",
	}

	param.QueryArgs.QueryArgumentMap = make(map[QueryArgumentMapKey]string)
	param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"ApplicationName"}] = filter

	msg.Body = param
	reply, err := n.transport.RequestReply(ctx, msg)
	if err != nil {
		return nil, err
	}

	body := reply.Body
	var b struct {
		ResultKind int32
		ResultList *struct {
			List []ApplicationQueryResult
		}
		ErrorCode int32
	}
	if err := serialization.Unmarshal(body, &b); err != nil {
		return nil, err
	}

	if b.ErrorCode != 0 {
		return nil, fmt.Errorf("GetApplicationList returns HResult %v", b.ErrorCode)
	}

	return b.ResultList.List, nil
}
