# cf-mssql-broker
A Go broker for MSSQL Service

## Summary

The cf-mssql-broker project implements and exposes the CF (Cloud Foundry) [Service Broker API](http://docs.cloudfoundry.org/services/api.html) to facilitate the management of a single Microsoft SQL Server instance. The minimum version supported is SQL Server 2012 Express.

For the provision operation of a CF service instance the broker will create a [contained database](https://msdn.microsoft.com/en-us/library/ff929071.aspx) on the targeted SQL Server.

For the binding operation the broker will create an SQL Server user and a randomly generated password in the contained database of the CF service instance.

The broker service does not need to save any state, thus it can be farmed or deployed on another box without any data migration. To keep track of provisioned instances and bindings it will use the IDs from Service Broker API in the database name and in the SQL Server user name.


## SQL Server config

### Enable TCP access for SQL Server
To provide SQL Access to Cloud Foundry applications, TCP has to be enabled for the SQL Server instance.
Use the SQL management studio to enable tcp or use the following doc to automate with PS: http://www.dbi-services.com/index.php/blog/entry/sql-server-2012-configuring-your-tcp-port-via-powershell

### Open SQL Server port

To configure or automate the firewall for SQL Server use the following PS example:

```sh
New-NetFirewallRule -DisplayName “SQL Server” -Direction Inbound –Protocol TCP –LocalPort 1433 -Action allow
```
 

### Enable Contained Database Authentication

Binding operations will create a user with a password only in the contained database. This is disabled by default in SQL Server 2012 and 2014. Use the following command to enable contained database authentication:

```sh
SQLCmd -S .\sqlexpress  -Q "EXEC sp_configure 'contained database authentication', 1; reconfigure;"
```

### Tips and Tricks

SQL Server can be installed with choco (https://chocolatey.org/):
choco install mssqlserver2012express

## Configuration

cf_mssql_broker_config.json is the default configuration file. The config file can be overridden with the following flag: -config=/new/path/config.json

The `servedMssqlBindingHostname` and `servedMssqlBindingPort` properties need to be changed for every installation. They are the hostname and port that are sent to the CF applications, and need to be accessible from the CF application network. NOTE: Do not change this value on an existing mssql broker with active bindings. If this is necessary, extra migration steps need to be taken for the existing bindings in the CF's Cloud Controller.

`logLevel` will set the logging level. Accepted levels: "debug", "info", "error", and "fatal".

The `brokerGoSqlDriver` and `brokerMssqlConnection` are settings that the broker uses to connect to the mssql instance. `brokerGoSqlDriver` can be "odbc" (recommended https://github.com/alexbrainman/odbc/) or "mssql" (experimental https://github.com/denisenkom/go-mssqldb). `brokerMssqlConnection` is a key-value JSON object that is 
converted into a connection string (e.g. {"server":"localhost","port":1433} is converted to  "server=localhost;port=1433") consumed by ODBC or mssql go library.
Example for a local trusted `brokerMssqlConnection` with ODBC driver:
	{
		"server":   "localhost\\sqlexpress",
		"database": "master",
		"driver": 	"sql server",
		"trusted_connection": "yes"
	}
	
`listeningAddr` and `brokerCredentials` are used for the brokers http server. The CF CloudController will use this setting to connect to the broker.

`dbIdentifierPrefix` is a string that is appended at the beginning of the instance ID for the SQL Server database name, and at the beginning of the binding id for the SQL Server user name. This will allow operators to easily identify the databases managed by a particular mssql broker. Do not change this value on a existing mssql broker with active instances.

`serviceCatalog` is a JSON object using the CF Service API catalog format and is sent to the Cloud Controller to identify the service name and plans, and provide a description to the user about the service. To add more mssql brokers to the same CF cluster will require the following changes: 
 > unique "name" for the service
 > unique "id" for the service
 > unique "id" for the plan

## Building and running

Setup you GOPATH env variable

```sh
go get -u -v github.com/tools/godep
go get github.com/cloudfoundry-incubator/cf-mssql-broker

cd $GOPATH/src/github.com/cloudfoundry-incubator/cf-mssql-broker # cd $env:GOPATH/src/github.com/cloudfoundry-incubator/cf-mssql-broker

godep restore
go build

# change the required values from the reference config file (cf_mssql_broker_config.json)
cf-mssql-broker -config=cf_mssql_broker_config.json
```

### Update dependencies

```sh
cd $GOPATH/src/github.com/cloudfoundry-incubator/cf-mssql-broker

# To update all packages:
go get -u -v go get github.com/cloudfoundry-incubator/cf-mssql-broker/...
godep update ...

# Or to update a specific package (e.g. odbc package):
go get -u -v github.com/alexbrainman/odbc
godep update github.com/alexbrainman/odbc

git add Godeps/*
```

## Using the broker with Curl REST calls

### Provision Instance

```sh
curl http://username:password@localhost:3000/v2/service_instances/instance1 -d '{ "service_id":  "b6844738-382b-4a9e-9f80-2ff5049d512f", "plan_id":           "fb740fd7-2029-467a-9256-63ecd882f11c",  "organization_guid": "org-guid-here", "space_guid":        "space-guid-here" }' -X PUT -H "X-Broker-API-Version: 2.4" -H "Content-Type: application/json"
```

### Bind Service Instance

```sh
curl http://username:password@localhost:3000/v2/service_instances/instance1/service_bindings/binding1 -d '{  "plan_id":        "plan-guid-here",  "service_id":     "service-guid-here",  "app_guid":       "app-guid-here"}' -X PUT -H "X-Broker-API-Version: 2.4" -H "Content-Type: application/json"
```

### Unbind Service Instance

```sh
curl 'http://username:password@localhost:3000/v2/service_instances/instance1/service_bindings/binding1?service_id=service-id-here&plan_id=plan-id-here' -X DELETE -H "X-Broker-API-Version: 2.4"
```

### Deprovision Instance

```sh
curl 'http://username:password@localhost:3000/v2/service_instances/instance1?service_id=b6844738-382b-4a9e-9f80-2ff5049d512f&plan_id=fb740fd7-2029-467a-9256-63ecd882f11c' -X DELETE -H "X-Broker-API-Version: 2.4"
```

## Windows Service installation

Use the following steps to install a windows service for the broker. Make sure you copy the binary and config file to "c:\cf-mssql-broker"

```sh
choco install nssm

$installDir = $env:systemdrive+'\cf-mssql-broker'
$installDir = 'C:\Users\schneids\workspace\gowp\src\cf-mssql-broker'

$exePath = Join-Path $installDir "cf-mssql-broker.exe"
$logPath = Join-Path $installDir "cf-mssql-broker.log"

mkdir -f $installDir

nssm install cf-mssql-broker $exePath

nssm set  cf-mssql-broker  AppStdout $logPath
nssm set  cf-mssql-broker  AppStderr $logPath

nssm start cf-mssql-broker
```

## Integrating into a Cloud Foundry deployemnt

You need admin access to a Cloud Foundry deployment to add a new service broker.

```sh
cf create-service-broker mssql-broker1 username password http://192.168.1.10:3000
cf enable-service-access mssql-dev
```

## Connecting to an external SQL Server

The broker service can run on the local SQL Server machine or on a remote machine (even as a CF app). It only needs to be able to send SQL queries/commands to the SQL Server. When the broker service is run on a remote location the `brokerMssqlConnection` has to be configured with the right IP, port, and credentials. You need to make sure that the network and firewall is setup so that the broker service has access to the SQL Server and that the credentials provided are authorized to create and drop databases.

Also, make sure the the CF applications that bind and connect to the SQL service database instances have network access to the configured `servedMssqlBindingHostname` and `servedMssqlBindingPort` entires. The following confiugations can affect what CF apps can reach in the network: Cloud Foundry security grous (i.e. cf security-groups), OpenStack/AWS sercurity groups for the DEAs/Cells and the SQL Server machines, Windows Firewall settings on the SQL Server machine, etc.

## Binding credentials exmaple

VCAP_SERVICES env variable for a CF application with a mssql service binding will contin the crednetials to the SQL Server. The folowing [credential fields](https://github.com/cloudfoundry-incubator/cf-mssql-broker/blob/master/mssql_binding_credentials.go) will be used:
 * "host" - IP address or host of the SQL Server
 * "port" - The listening TCP port number
 * "name" - Database name
 * "username" - User with credentials to the database
 * "password" - Password for the username
 * "connectionString" - Connection string that can be used directly in .NET applications, and may also work with as a base for ODBC or OleDb connection strings


Example:
```sh
cf env dotnetapp1
Getting env variables for app dotnetapp1 in org diego / space diego as admin...
OK
 
System-Provided:
{
 "VCAP_SERVICES": {
  "mssql-2014": [
   {
    "credentials": {
     "host": "10.0.0.93",
     "password": "DxdgJcdqzAbssMP7w_f7qsPtTlWklFhHHXLTw5_IlUI=qwerASF1234!@#$",
     "port": 1433,
	 "name": "cf-6536b7c1-6aa6-455f-9b54-fbe8de63053f",
     "username": "cf-6536b7c1-6aa6-455f-9b54-fbe8de63053f-856da771-d14b-4fad-a902-1eb02ff20c61",
	 "connectionString":"Address=10.0.0.93,1433;Database=cf-6536b7c1-6aa6-455f-9b54-fbe8de63053f;UID=cf-6536b7c1-6aa6-455f-9b54-fbe8de63053f-856da771-d14b-4fad-a902-1eb02ff20c61;PWD=DxdgJcdqzAbssMP7w_f7qsPtTlWklFhHHXLTw5_IlUI=qwerASF1234!@#$;"
    },
    "label": "mssql-2014",
    "name": "db1",
    "plan": "free",
    "tags": [
     "mssql",
     "relational"
    ]
   }
  ]
 }
}
...
```

How to use the bindings in a .NET app with cf-iis-buildpack:

The simplest way to use a SQL connection in the Web.config file inside the `<connectionStrings>` element:
```xml
<!-- replace 'db1' with your own Cloud Foundry service name to which the application is bound to -->
<add name="DefaultConnection" connectionString="{db1#connectionString}" providerName="System.Data.SqlClient"/>
```

Another way is to build the connection in the Web.config:
```xml
<add name="DefaultConnection" connectionString="Data Source={db1#hostname},{db1#port};Database={db1#name};User Id={db1#username};Password={db1#password}" providerName="System.Data.SqlClient"/>
```

Using the Odbc provider:
```xml
<add name="DefaultConnection" connectionString="Driver={SQL Server Native Client 11.0};{db1#connectionString}" providerName="System.Data.Odbc"/>
```

Using OleDb provider:
```xml
<add name="DefaultConnection" connectionString="Provider=SQLOLEDB;{db1#connectionString}" providerName="System.Data.OleDb"/>
```
