package main

import "fmt"

type MssqlBindingCredentials struct {
	Hostname         string `json:"hostname"`
	Port             int    `json:"port"`
	Name             string `json:"name"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	ConnectionString string `json:"connectionString"`
}

// References for connection strings:
// https://msdn.microsoft.com/en-us/library/system.data.sqlclient.sqlconnection.connectionstring(v=vs.110).aspx
// http://www.mono-project.com/docs/database-access/providers/sqlclient/
// http://freetds.schemamania.org/userguide/odbcconnattr.htm
// https://msdn.microsoft.com/en-us/library/ms130822.aspx
// This should be compatible with ADO.NET Sql connection string format.
// Also if posible, use only the subset that is also compatible with ODBC, FreeTds, and OleDb connection string.
// fmt template parameters: 1.address, 2.port, 3.database name, 4.username, 5. password
var connectionStringTemplate = "Address=%[1]v,%[2]v;Database=%[3]v;UID=%[4]v;PWD=%[5]v;"

func generateConnectionString(hostname string, port int, databaseName string, username string, password string) string {
	return fmt.Sprintf(connectionStringTemplate, hostname, port, databaseName, username, password)
}
