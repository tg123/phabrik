package transport

type MessageHeaderIdType uint16

const (
	MessageHeaderIdTypeInvalid MessageHeaderIdType = 0x8000
	// Transport level headers.
	MessageHeaderIdTypeActor        MessageHeaderIdType = 0x8001
	MessageHeaderIdTypeAction       MessageHeaderIdType = 0x8002
	MessageHeaderIdTypeMessageId    MessageHeaderIdType = 0x8003
	MessageHeaderIdTypeRelatesTo    MessageHeaderIdType = 0x8004
	MessageHeaderIdTypeExpectsReply MessageHeaderIdType = 0x8005
	MessageHeaderIdTypeRetry        MessageHeaderIdType = 0x8006
	MessageHeaderIdTypeFault        MessageHeaderIdType = 0x8007
	MessageHeaderIdTypeIdempotent   MessageHeaderIdType = 0x8008
	MessageHeaderIdTypeHighPriority MessageHeaderIdType = 0x8009
	MessageHeaderIdTypeREFrom       MessageHeaderIdType = 0x800a
	MessageHeaderIdTypeIpc          MessageHeaderIdType = 0x800b

	// Federation Protocol (Federation) headers
	MessageHeaderIdTypeFederationPartnerNode         MessageHeaderIdType = 0x800c
	MessageHeaderIdTypeFederationNeighborhoodRange   MessageHeaderIdType = 0x800d
	MessageHeaderIdTypeFederationNeighborhoodVersion MessageHeaderIdType = 0x800e
	MessageHeaderIdTypeFederationRoutingToken        MessageHeaderIdType = 0x800f
	MessageHeaderIdTypeRouting                       MessageHeaderIdType = 0x8010
	MessageHeaderIdTypeFederationTraceProbe          MessageHeaderIdType = 0x8011
	MessageHeaderIdTypeFederationTokenEcho           MessageHeaderIdType = 0x8012

	// Point to Point (PToP) Headers
	MessageHeaderIdTypePToP MessageHeaderIdType = 0x8013

	// Broadcast Headers
	MessageHeaderIdTypeBroadcast          MessageHeaderIdType = 0x8014
	MessageHeaderIdTypeBroadcastRange     MessageHeaderIdType = 0x8015
	MessageHeaderIdTypeBroadcastRelatesTo MessageHeaderIdType = 0x8016
	MessageHeaderIdTypeBroadcastStep      MessageHeaderIdType = 0x8017

	// Reliability
	MessageHeaderIdTypeGeneration MessageHeaderIdType = 0x8018

	// Replication
	MessageHeaderIdTypeReplicationActor     MessageHeaderIdType = 0x8019
	MessageHeaderIdTypeReplicationOperation MessageHeaderIdType = 0x801a
	MessageHeaderIdTypeCopyOperation        MessageHeaderIdType = 0x801b
	MessageHeaderIdTypeCompletedLSN         MessageHeaderIdType = 0x801c
	MessageHeaderIdTypeCopyContextOperation MessageHeaderIdType = 0x801d
	MessageHeaderIdTypeOperationAck         MessageHeaderIdType = 0x801e
	MessageHeaderIdTypeOperationError       MessageHeaderIdType = 0x801f

	// System Services (Common)
	MessageHeaderIdTypeFabricActivity      MessageHeaderIdType = 0x8020
	MessageHeaderIdTypeRequestInstance     MessageHeaderIdType = 0x8021
	MessageHeaderIdTypeSystemServiceFilter MessageHeaderIdType = 0x8022
	MessageHeaderIdTypeTimeout             MessageHeaderIdType = 0x8023

	// Naming Service
	MessageHeaderIdTypeCacheMode             MessageHeaderIdType = 0x8024
	MessageHeaderIdTypeClientProtocolVersion MessageHeaderIdType = 0x8025
	MessageHeaderIdTypeGatewayRetry          MessageHeaderIdType = 0x8026
	MessageHeaderIdTypePrimaryRecovery       MessageHeaderIdType = 0x8027

	// Cluster Manager Service
	MessageHeaderIdTypeForwardMessage MessageHeaderIdType = 0x8028

	// Security headers
	MessageHeaderIdTypeMessageSecurity MessageHeaderIdType = 0x8029

	// Query address header
	MessageHeaderIdTypeQueryAddress MessageHeaderIdType = 0x802a

	MessageHeaderIdTypeFabricCodeVersion MessageHeaderIdType = 0x802b

	MessageHeaderIdTypeServiceRoutingAgent      MessageHeaderIdType = 0x802c
	MessageHeaderIdTypeServiceRoutingAgentProxy MessageHeaderIdType = 0x802d

	// Reliable Messaging

	MessageHeaderIdTypeReliableMessagingSession          MessageHeaderIdType = 0x802e
	MessageHeaderIdTypeReliableMessagingSource           MessageHeaderIdType = 0x802f
	MessageHeaderIdTypeReliableMessagingTarget           MessageHeaderIdType = 0x8030
	MessageHeaderIdTypeReliableMessagingProtocolResponse MessageHeaderIdType = 0x8031
	MessageHeaderIdTypeReliableMessagingSessionParams    MessageHeaderIdType = 0x8032

	MessageHeaderIdTypeDeleteName MessageHeaderIdType = 0x8033

	MessageHeaderIdTypePartitionTarget    MessageHeaderIdType = 0x8034
	MessageHeaderIdTypeCustomClientAuth   MessageHeaderIdType = 0x8035
	MessageHeaderIdTypeNamingProperty     MessageHeaderIdType = 0x8036
	MessageHeaderIdTypeSecondaryLocations MessageHeaderIdType = 0x8037
	MessageHeaderIdTypeClientRole         MessageHeaderIdType = 0x8038

	MessageHeaderIdTypeMulticast        MessageHeaderIdType = 0x8039
	MessageHeaderIdTypeMulticastTargets MessageHeaderIdType = 0x803a

	MessageHeaderIdTypeFileUploadRequest MessageHeaderIdType = 0x803b
	MessageHeaderIdTypeFileSequence      MessageHeaderIdType = 0x803c

	MessageHeaderIdTypeServiceTarget MessageHeaderIdType = 0x803d

	MessageHeaderIdTypeUncorrelatedReply MessageHeaderIdType = 0x803e

	MessageHeaderIdTypeServiceDirectMessaging MessageHeaderIdType = 0x803f

	MessageHeaderIdTypeClientIdentity MessageHeaderIdType = 0x8040
	MessageHeaderIdTypeServerAuth     MessageHeaderIdType = 0x8041

	MessageHeaderIdTypeGlobalTimeExchange MessageHeaderIdType = 0x8041
	MessageHeaderIdTypeVoterStore         MessageHeaderIdType = 0x8042

	//Service Tcp Communication
	MessageHeaderIdTypeServiceLocationActor      MessageHeaderIdType = 0x8043
	MessageHeaderIdTypeTcpServiceMessageHeader   MessageHeaderIdType = 0x8044
	MessageHeaderIdTypeTcpClientIdHeader         MessageHeaderIdType = 0x8045
	MessageHeaderIdTypeServiceCommunicationError MessageHeaderIdType = 0x8046
	MessageHeaderIdTypeIsAsyncOperationHeader    MessageHeaderIdType = 0x8047

	MessageHeaderIdTypeSecurityNegotiation MessageHeaderIdType = 0x8048

	MessageHeaderIdTypeJoinThrottle MessageHeaderIdType = 0x8049

	// Replication
	MessageHeaderIdTypeReplicationOperationBody MessageHeaderIdType = 0x804a

	MessageHeaderIdTypeCreateComposeDeploymentRequest MessageHeaderIdType = 0x804b
	// To be compatible with v6.0 which was using 0x804c conflicted with FabricTransportMessageHeader. From v6.1 the header id would be 0x804d.
	MessageHeaderIdTypeUpgradeComposeDeploymentRequest_Compatibility MessageHeaderIdType = 0x804c
	//Fabric Transport V2
	MessageHeaderIdTypeFabricTransportMessageHeader    MessageHeaderIdType = 0x804c
	MessageHeaderIdTypeUpgradeComposeDeploymentRequest MessageHeaderIdType = 0x804d
	MessageHeaderIdTypeCreateVolumeRequest             MessageHeaderIdType = 0x804e
	MessageHeaderIdTypeFileUploadCreateRequest         MessageHeaderIdType = 0x804f
	MessageHeaderIdTypeFileTransferTransportDownload   MessageHeaderIdType = 0x8050
	MessageHeaderIdTypeFileTransferTransportUpload     MessageHeaderIdType = 0x8051
	MessageHeaderIdTypeFileTransferTransportError      MessageHeaderIdType = 0x8052

	MessageHeaderIdTypeFederationForwardMessaging MessageHeaderIdType = 0x8053
	MessageHeaderIdTypeFederationAgentRequest     MessageHeaderIdType = 0x8054

	MessageHeaderIdTypeSystemServiceTcpHeader MessageHeaderIdType = 0x8055
	MessageHeaderIdTypeTransportRoutingHeader MessageHeaderIdType = 0x8056
)
