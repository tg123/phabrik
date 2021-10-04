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

type FabriQueryResultKind int32

const (
	FabriQueryResultKindInvalid FabriQueryResultKind = 0x00
	FabriQueryResultKindItem    FabriQueryResultKind = 0x01
	FabriQueryResultKindList    FabriQueryResultKind = 0x02
)

type FabricServicePartitionKind int64

const (
	FabricServicePartitionKindInvalid    = 0x0
	FabricServicePartitionKindSingleton  = 0x1
	FabricServicePartitionKindInt64Range = 0x2
	FabricServicePartitionKindNamee      = 0x3
)

type FabricServiceKind int64

const (
	FabricServiceKindInvalid   = 0
	FabricServiceKindStateless = 0x1
	FabricServiceKindStatefull = 0x2
)

type FabricReplicaRole int64

const (
	FabricReplicaRoleUnknown          = 0x0
	FabricReplicaRoleNone             = 0x1
	FabricReplicaRolePrimary          = 0x2
	FabricReplicaRoleIdleSecondary    = 0x3
	FabricReplicaRoleActiveSecondary  = 0x4
	FabricReplicaRoleIdelAuxiliary    = 0x5
	FabricReplicaRoleActiveAuxiliary  = 0x6
	FabricReplicaRolePrimaryAuxiliary = 0x7
)

type FabricReplicaStatus int64

const (
	FabricReplicaStatusInvalid   = 0
	FabricReplicaStatusInBuild   = 0x1
	FabricReplicaStatusStandby   = 0x2
	FabricReplicaStatusReady     = 0x3
	FabricReplicaStatusDown      = 0x4
	FabricReplicaStatusDropped   = 0x5
	FabricReplicaStatusCompleted = 0x6
)

type FabricHealthState int64

const (
	FabricHealthStateInvalid = 0
	FabricHealthStateOK      = 0x1
	FabricHealthStateWarning = 0x2
	FabricHealthStateError   = 0x3
	FabricHealthStateUnknown = 0xffff
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

func NewNamingMessage(action string) (*transport.Message, error) {
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
		Timeout: 2000 * time.Millisecond,
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

		reply, err := NewNamingMessage("ServiceNotificationReply")
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
	msg, err := NewNamingMessage("PingRequest")

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
		msg, err := NewNamingMessage("NotificationClientConnectionRequest")
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
		msg, err := NewNamingMessage("RegisterServiceNotificationFilterRequest")
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

type QueryArgumentMapKey struct {
	Key string
}

type QueryRequest struct {
	QueryArgs struct {
		QueryArgumentMap map[QueryArgumentMapKey]string
	}
	QueryName string
}

func NewQueryRequest(requestType string) *QueryRequest {
	qr := &QueryRequest{
		QueryName: requestType,
	}

	qr.QueryArgs.QueryArgumentMap = make(map[QueryArgumentMapKey]string)
	return qr
}

type ApplicationQueryResult struct {
	ApplicationName        common.Uri
	ApplicationTypeName    string
	ApplicationTypeVersion string
	ApplicationStatus      int64
	HealthState            int64
	ApplicationParameters  map[string]string
}

// GetApplicationList enumerates the applications
func (n *NamingClient) GetApplicationList(ctx context.Context, filter string) ([]ApplicationQueryResult, error) {

	msg, err := NewNamingMessage("QueryRequest")
	if err != nil {
		return nil, err
	}

	param := NewQueryRequest("GetApplicationList")
	param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"ApplicationName"}] = filter

	msg.Body = param
	reply, err := n.transport.RequestReply(ctx, msg)
	if err != nil {
		return nil, err
	}

	body := reply.Body
	var b struct {
		ResultKind FabriQueryResultKind
		ResultList *struct {
			List []ApplicationQueryResult
		}
		ErrorCode int32
	}
	if err := serialization.Unmarshal(body, &b); err != nil {
		return nil, err
	}

	if b.ErrorCode != 0 || b.ResultKind != FabriQueryResultKindList {
		return nil, fmt.Errorf("GetApplicationList returns HResult %v", b.ErrorCode)
	}

	return b.ResultList.List, nil
}

type ServiceQueryResult struct {
	ServiceKind       int32
	ServiceName       common.Uri
	ServiceTypeName   string
	ManifestVersion   string
	HasPersistedState bool
	HealthState       int64
	ServiceStatus     int64
	IsServiceGroup    bool
}

// GetServiceList enumerates the services for an application
func (n *NamingClient) GetServiceList(ctx context.Context, applicationName string) ([]ServiceQueryResult, error) {

	msg, err := NewNamingMessage("QueryRequest")
	if err != nil {
		return nil, err
	}

	param := NewQueryRequest("GetApplicationServiceList")
	param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"ApplicationName"}] = applicationName

	msg.Body = param
	reply, err := n.transport.RequestReply(ctx, msg)
	if err != nil {
		return nil, err
	}

	body := reply.Body
	var b struct {
		ResultKind FabriQueryResultKind
		ResultList *struct {
			List []ServiceQueryResult
		}
		ErrorCode int32
	}
	if err := serialization.Unmarshal(body, &b); err != nil {
		return nil, err
	}

	if b.ErrorCode != 0 || b.ResultKind != FabriQueryResultKindList {
		return nil, fmt.Errorf("GetApplicationList returns HResult %v", b.ErrorCode)
	}

	return b.ResultList.List, nil
}

type ServiceType struct {
	ServiceTypeDescription ServiceTypeDescription `json:"ServiceTypeDescription"`
	ServiceManifestVersion string                 `json:"ServiceManifestVersion"`
	ServiceManifestName    string                 `json:"ServiceManifestName"`
	IsServiceGroup         bool                   `json:"IsServiceGroup"`
}

// ServiceTypeDescription Service Type Description
type ServiceTypeDescription struct {
	IsStateful               bool                                `json:"IsStateful"`
	ServiceTypeName          string                              `json:"ServiceTypeName"`
	PlacementConstraints     string                              `json:"PlacementConstraints"`
	HasPersistedState        bool                                `json:"HasPersistedState"`
	UseImplicitHost          bool                                `json:"UseImplicitHost"`
	Extensions               []KeyValuePair                      `json:"Extensions"`
	LoadMetrics              []LoadMetrics                       `json:"LoadMetrics"`
	ServicePlacementPolicies []ServicePlacementPolicyDescription `json:"ServicePlacementPolicies"`
}

type LoadMetrics struct {
	Name                 string `json:"Name"`
	Weight               uint32 `json:"Weight"`
	PrimaryDefaultLoad   uint32 `json:"PrimaryDefaultLoad"`
	SecondaryDefaultLoad uint32 `json:"SecondaryDefaultLoad"`
	DefaultLoad          uint32 `json:"DefaultLoad"`
	AuxiliaryDefaultLoad uint32 `json:"AuxiliaryDefaultLoad"`
}

type ServicePlacementPolicyDescription struct {
	Type       uint32 `json:"Type"`
	DomainName string `json:"DomainName"`
}

type KeyValuePair struct {
	Key   string `json:"Key"`
	Value string `json:"Value"`
}

// GetServiceTypeList enumerates the ServiceTypes for an application type and version
func (n *NamingClient) GetServiceTypeList(ctx context.Context, applicationTypeName, applicationTypeVersion, serviceTypeNameFilter string) ([]ServiceType, error) {

	msg, err := NewNamingMessage("QueryRequest")
	if err != nil {
		return nil, err
	}

	param := NewQueryRequest("GetServiceTypeList")
	param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"ApplicationTypeName"}] = applicationTypeName
	param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"ApplicationTypeVersion"}] = applicationTypeVersion

	if serviceTypeNameFilter != "" {
		param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"ServiceTypeNameFilter"}] = serviceTypeNameFilter
	}

	msg.Body = param
	reply, err := n.transport.RequestReply(ctx, msg)
	if err != nil {
		return nil, err
	}

	body := reply.Body
	var b struct {
		ResultKind FabriQueryResultKind
		ResultList *struct {
			List []ServiceType
		}
		ErrorCode int32
	}
	if err := serialization.Unmarshal(body, &b); err != nil {
		return nil, err
	}

	if b.ErrorCode != 0 || b.ResultKind != FabriQueryResultKindList {
		return nil, fmt.Errorf("GetServiceTypeList returns hr %v", b.ErrorCode)
	}

	return b.ResultList.List, nil
}

type PartitionInformation struct {
	PartitionKind FabricServicePartitionKind
	PartitionId   serialization.GUID
	PartitionName string
	LowKey        int64
	HighKey       int64
}

type PartitionQueryResult struct {
	ServiceKind           int32
	PartitionInformation  PartitionInformation
	InstanceCount         int32
	MinInstanceCount      int32
	MinInstancePercentage int32
	TargetReplicaSetSize  int32
	MinReplicaSetSize     int32
	HealthState           int32
	//PartitionStatus       int32
	//LastQuorumLossDurationInSeconds
	//CurrentConfigurationEpoch <--
	//AuxiliaryReplicaCount
}

// GetServicePartitionList enumerates the partitions for a service
func (n *NamingClient) GetServicePartitionList(ctx context.Context, serviceName string) ([]PartitionQueryResult, error) {

	msg, err := NewNamingMessage("QueryRequest")
	if err != nil {
		return nil, err
	}

	param := NewQueryRequest("GetServicePartitionList")
	param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"ServiceName"}] = serviceName

	//if serviceTypeNameFilter != "" {
	//	param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"ServiceTypeNameFilter"}] = serviceTypeNameFilter
	//}

	msg.Body = param
	reply, err := n.transport.RequestReply(ctx, msg)
	if err != nil {
		return nil, err
	}

	body := reply.Body
	var b struct {
		ResultKind FabriQueryResultKind
		//ContinuationToken string
		ResultList *struct {
			List []PartitionQueryResult
		}
		ErrorCode int32
	}
	if err := serialization.Unmarshal(body, &b); err != nil {
		return nil, err
	}

	if b.ErrorCode != 0 || b.ResultKind != FabriQueryResultKindList {
		return nil, fmt.Errorf("GetServicePartitionList returns hr %v", b.ErrorCode)
	}

	return b.ResultList.List, nil
}

type Replica struct {
	ServiceKind   FabricServiceKind
	ReplicaId     int64
	ReplicaRole   FabricReplicaRole
	InstanceId    int64
	ReplicaStatus FabricReplicaStatus
	HealthState   FabricHealthState
	//Address                      string
	//NodeName                     string
	//LastInBuildDurationInSeconds int64
}

// GetServicePartitionReplicaList enumerates the replicas for a partitionId
func (n *NamingClient) GetServicePartitionReplicaList(ctx context.Context, partitionId string) ([]Replica, error) {

	msg, err := NewNamingMessage("QueryRequest")
	if err != nil {
		return nil, err
	}

	param := NewQueryRequest("GetServicePartitionReplicaList")
	param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"PartitionId"}] = partitionId

	//if serviceTypeNameFilter != "" {
	//	param.QueryArgs.QueryArgumentMap[QueryArgumentMapKey{"ServiceTypeNameFilter"}] = serviceTypeNameFilter
	//}

	msg.Body = param
	reply, err := n.transport.RequestReply(ctx, msg)
	if err != nil {
		return nil, err
	}

	body := reply.Body
	var b struct {
		ResultKind FabriQueryResultKind
		//ContinuationToken string
		ResultList *struct {
			List []Replica
		}
		ErrorCode int32
	}
	if err := serialization.Unmarshal(body, &b); err != nil {
		return nil, err
	}

	if b.ErrorCode != 0 || b.ResultKind != FabriQueryResultKindList {
		return nil, fmt.Errorf("GetServicePartitionReplicaList returns hr %v", b.ErrorCode)
	}

	return b.ResultList.List, nil
}

type Property struct {
	Data           string
	Name           string
	PropertyString int16
	Size           int64
	CustomType     string
}

type ContinuationToken struct {
	LastProperty      string
	PropertiesVersion int64
	IsValid           bool
}

func (n *NamingClient) EnumerateProperties(ctx context.Context, serviceName string) ([]Property, error) {

	msg, err := NewNamingMessage("EnumeratePropertiesRequest")
	if err != nil {
		return nil, err
	}

	param := &struct {
		serviceUri        common.Uri
		continuationToken *ContinuationToken
		includeValues     bool
	}{
		serviceUri: common.Uri{
			Type:         common.UriTypeAbsolute,
			Scheme:       "fabric",
			Authority:    "",
			HostType:     common.UriHostTypeNone,
			Host:         "",
			Port:         -1,
			Path:         "/pinger0/PingerService",
			PathSegments: []string{"pinger0", "PingerService"},
		},
		includeValues: true,
	}

	msg.Body = param
	reply, err := n.transport.RequestReply(ctx, msg)
	if err != nil {
		return nil, err
	}

	body := reply.Body
	var b struct {
		Result            int32
		Properties        []Property
		ContinuationToken string
	}
	if err := serialization.Unmarshal(body, &b); err != nil {
		return nil, err
	}

	if b.Result != 0 {
		return nil, fmt.Errorf("EnumerateProperties returns HResult %v", b.Result)
	}

	return b.Properties, nil
}
