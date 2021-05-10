package naming

import (
	"context"
	"fmt"
	"time"

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
	Transport *transport.Client
}

type GatewayDescription struct {
	Address      string
	NodeInstance NodeInstance
	NodeName     string
}

func (n *NamingClient) Ping() (*GatewayDescription, error) {
	msg, err := newNamingMessage("PingRequest")

	if err != nil {
		return nil, err
	}

	reply, err := n.Transport.RequestReply(context.TODO(), msg)
	if err != nil {
		return nil, err
	}

	body, ok := reply.Body.([]byte)
	if !ok {
		return nil, fmt.Errorf("body not a []byte")
	}
	switch reply.Headers.Action {
	case "PingRequest":
		var b struct {
			GatewayDescription GatewayDescription
		}
		if err := serialization.Unmarshal(body, &b); err != nil {
			return nil, err
		}

		return &b.GatewayDescription, nil
	case "ClientOperationFailure":
		var b struct {
			ErrorCode int32
		}
		if err := serialization.Unmarshal(body, &b); err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("ClientOperationFailure HResult: %v", b.ErrorCode)
	default:
		return nil, fmt.Errorf("unsupported action %v MessageActorTypeNamingGateway", reply.Headers.Action)
	}
}

type ApplicationQueryResult struct {
	ApplicationName        Uri
	ApplicationTypeName    string
	ApplicationTypeVersion string
	ApplicationStatus      int32
	HealthState            int32
	ApplicationParameters  map[string]string
}

func (n *NamingClient) GetApplicationList(filter string) ([]ApplicationQueryResult, error) {

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
	reply, err := n.Transport.RequestReply(context.TODO(), msg)
	if err != nil {
		return nil, err
	}

	body, ok := reply.Body.([]byte)
	if !ok {
		return nil, fmt.Errorf("body not a []byte")
	}
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
