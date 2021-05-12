# Phabrik

[![](https://pkg.go.dev/badge/github.com/tg123/phabrik?status.svg)](https://pkg.go.dev/github.com/tg123/phabrik)

**Yet another [Service Fabric](https://azure.microsoft.com/en-us/services/service-fabric/) Client:**

Unlike other COM+ clients, Go <https://github.com/tg123/fabric> or .net <https://www.nuget.org/packages/Microsoft.ServiceFabric/>, the implementation is 100% native Go code without any interop.

**More than Fabric Client:**

The Transport layer can act as Server role as well, accepting any queries from normal Fabric Client.

## Examples

### [powershellserver](./examples/powershellserver)

A fake Service Fabric server which accepts SF Powershell Client

Start server

```
powershellserver.exe 127.0.0.1:9998 123bdacdcdfb2c7b250192c6078e47d1e1db119b
```

Connect from Powershell (No Command Implemented)

```
Connect-ServiceFabricCluster 127.0.0.1:9998  -FindValue "123bdacdcdfb2c7b250192c6078e47d1e1db119b" -FindType FindByThumbprint -X509Credential -ServerCertThumbprint "123bdacdcdfb2c7b250192c6078e47d1e1db119b" -StoreLocation CurrentUser -StoreName My
True


FabricClientSettings         : {
                               ClientFriendlyName                   : PowerShell-3b2227e3-45f7-4e16-afab-efdfbef1a9dc
                               PartitionLocationCacheLimit          : 100000
                               PartitionLocationCacheBucketCount    : 1024
                               ServiceChangePollInterval            : 00:02:00
                               ConnectionInitializationTimeout      : 00:00:02
                               KeepAliveInterval                    : 00:00:20
                               ConnectionIdleTimeout                : 00:00:00
                               HealthOperationTimeout               : 00:02:00
                               HealthReportSendInterval             : 00:00:00
                               HealthReportRetrySendInterval        : 00:00:30
                               NotificationGatewayConnectionTimeout : 00:00:30
                               NotificationCacheUpdateTimeout       : 00:00:30
                               AuthTokenBufferSize                  : 4096
                               }
GatewayInformation           : {
                               NodeAddress                          : 127.0.0.1:9998
                               NodeId                               : dc87bf4b3176cc286c5d707132f62a9
                               NodeInstanceId                       : 1000
                               NodeName                             : NodeName
                               }
FabricClient                 : System.Fabric.FabricClient
ConnectionEndpoint           : {127.0.0.1:9998}
SecurityCredentials          : System.Fabric.X509Credentials
AzureActiveDirectoryMetadata :
```

### [query](./examples/query)

A fake client list all application from a Service Fabric endpoint

```
query.exe test.southcentralus.cloudapp.azure.com:19000 123bdacdcdfb2c7b250192c6078e47d1e1db119b
Remote thumbprint 42a9de9c9deaadd96057932bef6d4b9299ea5f8d
2021/05/12 10:44:20 Connected, Gateway info: &{10.0.0.6:19000 {a524682b4ceb893541e862483db07d22 132647381747950940} FE29236_2}
2021/05/12 10:44:20 Applications:  [{{1 fabric  0  -1 /testapp   [testapp]} testappType 1.0.0 1 65535 map[]}]
```


## Packages 

The packages are mapping from service fabric source code and exporting Go API

### Serialization
_Service Fabric Code: <https://github.com/microsoft/service-fabric/tree/master/src/prod/shared/serialization/>_

Package `serialization` implements encoding and decoding of Service Fabric binary protocol

### Transport
_Service Fabric Code: <https://github.com/microsoft/service-fabric/tree/master/src/prod/src/Transport>_

Package `transport` implements tcp/networking protocol with Service Fabric endpoints

