package lease

import "time"

type LeaseMessageType int32

const (
	LeaseMessageTypeLeaseRequest LeaseMessageType = iota
	LeaseMessageTypeLeaseResponse
	LeaseMessageTypePingRequest
	LeaseMessageTypePingResponse
	LeaseMessageTypeForwardRequest
	LeaseMessageTypeForwardResponse
	LeaseMessageTypeRelayRequest
	LeaseMessageTypeRelayResponse
)

type messageBody struct {
	SubjectEstablishPendingList  []string
	SubjectFailedPendingList     []string
	MonitorFailedPendingList     []string
	SubjectPendingAcceptedList   []string
	SubjectPendingRejectedList   []string
	SubjectFailedAcceptedList    []string
	MonitorFailedAcceptedList    []string
	SubjectTerminatePendingList  []string
	SubjectTerminateAcceptedList []string
}

type Message struct {
	Identifier               int64
	Type                     LeaseMessageType
	LeaseInstance            int64
	RemoteLeaseAgentInstance int64
	Duration                 time.Duration
	Expiration               time.Duration
	LeaseSuspendDuration     time.Duration
	ArbitrationDuration      time.Duration
	IsTwoWayTermination      bool
	MessageListenEndpoint    string
	messageBody
}
