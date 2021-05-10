# Phabrik

[![](https://pkg.go.dev/badge/github.com/tg123/phabrik?status.svg)](https://pkg.go.dev/github.com/tg123/phabrik)

**Yet another [Service Fabric](https://azure.microsoft.com/en-us/services/service-fabric/) Client:**

Unlike other COM+ clients, Go <https://github.com/tg123/fabric> or .net <https://www.nuget.org/packages/Microsoft.ServiceFabric/>, the implementation is 100% native Go code without any interop.

**More than Fabric Client:**

The Transport layer can act as Server role as well, accepting any queries from normal Fabric Client.

## Usage and Packages 

The packages are mapping from service fabric source code and exporting Go API

### Serialization
_Service Fabric Code: <https://github.com/microsoft/service-fabric/tree/master/src/prod/shared/serialization/>_

Package `serialization` implements encoding and decoding of Service Fabric binary protocol

### Transport
_Service Fabric Code: <https://github.com/microsoft/service-fabric/tree/master/src/prod/src/Transport>_

Package `transport` implements tcp/networking protocol with Service Fabric endpoints

