package transport

type MessageActorType int64

const (
	MessageActorTypeEmpty MessageActorType = 0

	// Transport
	MessageActorTypeTransport MessageActorType = 1

	// Federation
	MessageActorTypeFederation MessageActorType = 2
	MessageActorTypeRouting    MessageActorType = 3

	// Cluster Manager
	MessageActorTypeCM MessageActorType = 4

	// Naming
	MessageActorTypeNamingGateway      MessageActorType = 5
	MessageActorTypeNamingStoreService MessageActorType = 6

	// Hosting
	MessageActorTypeApplicationHostManager MessageActorType = 7
	MessageActorTypeApplicationHost        MessageActorType = 8
	MessageActorTypeFabricRuntimeManager   MessageActorType = 9

	// Failover
	MessageActorTypeFMM             MessageActorType = 10
	MessageActorTypeFM              MessageActorType = 11
	MessageActorTypeRA              MessageActorType = 12
	MessageActorTypeRS              MessageActorType = 13
	MessageActorTypeServiceResolver MessageActorType = 14

	// Hosting subsystem
	MessageActorTypeHosting MessageActorType = 15

	// HealthManager
	MessageActorTypeHM MessageActorType = 16

	// Infrastructure Service
	MessageActorTypeServiceRoutingAgent MessageActorType = 17
	MessageActorTypeIS                  MessageActorType = 18

	//Fabric Activator
	MessageActorTypeFabricActivator       MessageActorType = 19
	MessageActorTypeFabricActivatorClient MessageActorType = 20

	// Transport
	MessageActorTypeIpc MessageActorType = 21

	// FileStoreService
	MessageActorTypeFileStoreService MessageActorType = 22

	// TokenValidationService
	MessageActorTypeTvs MessageActorType = 23

	// Repair Manager
	MessageActorTypeRM MessageActorType = 24

	MessageActorTypeFileSender          MessageActorType = 25
	MessageActorTypeFileReceiver        MessageActorType = 26
	MessageActorTypeFileTransferClient  MessageActorType = 27
	MessageActorTypeFileTransferGateway MessageActorType = 28

	MessageActorTypeTransportSendTarget    MessageActorType = 29
	MessageActorTypeEntreeServiceProxy     MessageActorType = 30
	MessageActorTypeEntreeServiceTransport MessageActorType = 31
	MessageActorTypeHostedServiceActivator MessageActorType = 32

	MessageActorTypeNM                   MessageActorType = 33
	MessageActorTypeDirectMessagingAgent MessageActorType = 34

	MessageActorTypeSecurityContext           MessageActorType = 35
	MessageActorTypeServiceCommunicationActor MessageActorType = 36

	MessageActorTypeRestartManager       MessageActorType = 37
	MessageActorTypeRestartManagerClient MessageActorType = 38

	// FaultAnalysisService
	MessageActorTypeFAS MessageActorType = 39

	// TestabilityAgent
	MessageActorTypeTestabilitySubsystem MessageActorType = 40

	// UpgradeOrchestrationService
	MessageActorTypeUOS MessageActorType = 41

	// Backup Restore Agent.
	MessageActorTypeBA  MessageActorType = 42
	MessageActorTypeBRS MessageActorType = 43
	MessageActorTypeBAP MessageActorType = 44

	// Fabric Container Activator Service
	MessageActorTypeContainerActivatorService       MessageActorType = 45
	MessageActorTypeContainerActivatorServiceClient MessageActorType = 46

	MessageActorTypeResourceMonitor MessageActorType = 47

	// Central Secret Service
	MessageActorTypeCSS MessageActorType = 48

	// NetworkInventoryService
	MessageActorTypeNetworkInventoryService MessageActorType = 49

	// NetworkInventoryAgent
	MessageActorTypeNetworkInventoryAgent MessageActorType = 50

	// GatewayResourceManager Service
	MessageActorTypeGatewayResourceManager MessageActorType = 51

	// FederationAgent
	MessageActorTypeFederationAgent MessageActorType = 54

	// FederationAgentProxy
	MessageActorTypeFederationProxy MessageActorType = 55

	// SystemServiceTcpConnection
	MessageActorTypeSystemServiceTcpConnection MessageActorType = 56

	MessageActorTypeNamingGatewayService MessageActorType = 57

	MessageActorTypeSystemServiceConfigSetting MessageActorType = 58

	MessageActorTypeSystemCache MessageActorType = 59

	MessageActorTypeEndValidEnum MessageActorType = 60

	// !!! Please add new actor values above this !!!
	MessageActorTypeFirstValidEnum MessageActorType = MessageActorTypeEmpty
	MessageActorTypeLastValidEnum  MessageActorType = MessageActorTypeEndValidEnum - 1

	// Test range
	MessageActorTypeWindowsFabricTestApi MessageActorType = 0xFFFF
	MessageActorTypeGenericTestActor     MessageActorType = 0x10000
	MessageActorTypeGenericTestActor2    MessageActorType = 0x10001
	MessageActorTypeDistributedSession   MessageActorType = 0x10002
	MessageActorTypeIpcTestActor1        MessageActorType = 0x10003
	MessageActorTypeIpcTestActor2        MessageActorType = 0x10004
)
