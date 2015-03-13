# cf-mssql-broker
A Go broker for MSSQL Service

## Summary

The cf-mssql-broker project implements and exposes the CF (Cloud Foundy) Service Broker API (http://docs.cloudfoundry.org/services/api.html) to facilitate the managemnt of a single Microsoft SQL Server instance. The minimum version supported is SQL Server 2012 Express.

For the provision operation of a CF service instance the broker will create a contained database (https://msdn.microsoft.com/en-us/library/ff929071.aspx) on the targeted SQL Server.

For the binding operation the broker will create an SQL Server user and a ramdomly genrated password in the contained database of the CF service instance.

The broker serive does not need to have save any state, thus it can be farmed or deployed on another box without any data migration. To keep track of provisioned instances and bindings it will use the IDs from Service Broker API in the database name and in the SQL Server user name.



## SQL Server config

### Enable TCP access for SQL Server
To provide SQL Access to Cloud Foundry applications, TCP has to be enabled for the SQL Server instance.
Use the sql management studio to enable tcp or use the following doc to automate with PS: http://www.dbi-services.com/index.php/blog/entry/sql-server-2012-configuring-your-tcp-port-via-powershell

### Open SQL Server port

To configure or automate the firewall for SQL Server use the following PS example:

New-NetFirewallRule -DisplayName “SQL Server” -Direction Inbound –Protocol TCP –LocalPort 1433 -Action allow
 

### Enable Contained Database Authentication

Binding operations will create a user with a password only in the contained database. This is disabled by 
default in SQL Server 2012 and 2014. Use the following commmand to enable contained database authentication:

SQLCmd -S .\sqlexpress  -Q "EXEC sp_configure 'contained database authentication', 1; reconfigure;"

### Tips and Tricks

SQL Server can be installed with choco (https://chocolatey.org/):
choco install mssqlserver2012express

## Configuration

cf_mssql_broker_config.json is the default configuration file. The config file can be overridden with
the following flag: -config=/new/path/config.json

The servedMssqlBindingHostname and servedMssqlBindingPort need to be changed for every installetion.
They are the hostname and port that are sent to the CF applicatoins, and need to be accessible from the CF application network. NOTE: Do not change this value on a existing mssql broker with active bindings.
It this is necessary, extra migrations steps need to be taken for the exsing bindings in the CF's Cloud Controller.

The brokerGoSqlDriver and brokerMssqlConnection are settings that the broker uses to connect to the mssql instance. brokerGoSqlDriver can be odbc (recommanded https://code.google.com/p/odbc/) for mssql (experimental https://github.com/denisenkom/go-mssqldb). brokerMssqlConnection is a map that is 
converted into a connection string (e.g. "server=localhost;port=1433") consumed by odbc or mssql go library.
Exmaple for a local trusted brokerMssqlConnection with odbc driver:
	{
		"server":   "localhost\\sqlexpress",
		"database": "master",
		"driver": 	"sql server",
		"trusted_connection": "yes"
	}
	
listeningAddr and brokerCredentials are used for the brokers http server. The CF CloudController will use this setting to connect to the broker.

dbIdentifierPrefix appended at the begining of the instance ID for the SQL Server database name, and it is appended at the begining of the binding id for the SQL Server user name. This will allow an adim to easily idetify the databases managed by a particular mssql broker.

serviceCatalog JSON is sent to the Cloud Controller to identify the service name and plans, and provide additional description to the user.

## Building and running

Setup you GOPATH env variable

go get github.com/tools/godep
go get github.com/hpcloud/cf-mssql-broker

cd $GOPATH/src/github.com/hpcloud/cf-mssql-broker # cd $env:GOPATH/src/github.com/hpcloud/cf-mssql-broker

godep restore
go build
cf-mssql-broker -config=cf_mssql_broker_config.json


## Using the broker with Curl REST calls

### Provision Instance

curl http://username:password@localhost:3000/v2/service_instances/instance1 -d '{ "service_id":  "b6844738-382b-4a9e-9f80-2ff5049d512f", "plan_id":           "fb740fd7-2029-467a-9256-63ecd882f11c",  "organization_guid": "org-guid-here", "space_guid":        "space-guid-here" }' -X PUT -H "X-Broker-API-Version: 2.4" -H "Content-Type: application/json"

### Bind Service Instance

curl http://username:password@localhost:3000/v2/service_instances/instance1/service_bindings/binding1 -d '{  "plan_id":        "plan-guid-here",  "service_id":     "service-guid-here",  "app_guid":       "app-guid-here"}' -X PUT -H "X-Broker-API-Version: 2.4" -H "Content-Type: application/json"

### Unbind Service Instance

curl 'http://username:password@localhost:3000/v2/service_instances/instance1/service_bindings/binding1?service_id=service-id-here&plan_id=plan-id-here' -X DELETE -H "X-Broker-API-Version: 2.4"

### Deprovision Instance
curl 'http://username:password@localhost:3000/v2/service_instances/instance1?service_id=b6844738-382b-4a9e-9f80-2ff5049d512f&plan_id=fb740fd7-2029-467a-9256-63ecd882f11c' -X DELETE -H "X-Broker-API-Version: 2.4"

## Windows Service installation

Use the following steps to install a windows service for the broker. Make sure you copy the binary 
and config file to "c:\cf-mssql-broker"

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

## Integrating into a Cloud Foundry deployemnt

You need admin access to a Cloud Foundry deployment to add a new service broker.

cf create-service-broker mssql-broker1 username password http://192.168.1.10:3000
cf enable-service-access mssql-dev
