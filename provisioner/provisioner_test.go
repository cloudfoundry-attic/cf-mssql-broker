package provisioner

import (
	"database/sql"
	"github.com/pivotal-golang/lager/lagertest"
	"testing"
)

var logger *lagertest.TestLogger = lagertest.NewTestLogger("mssql-provisioner")
var mssqlPars = map[string]string{
	"driver":             "sql server",
	"server":             "(local)\\sqlexpress",
	"database":           "master",
	"trusted_connection": "yes",
}

func TestCreateDatabase(t *testing.T) {
	dbName := "cf-broker-testing.create-db"

	sqlClient, err := sql.Open("odbc", buildConnectionString(mssqlPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, mssqlPars)
	mssqlProv.Init()

	// Act
	err = mssqlProv.CreateDatabase("cf-broker-testing.create-db")

	// Assert
	if err != nil {
		t.Errorf("Database create error, %v", err)
	}
	defer sqlClient.Exec("drop database [" + dbName + "]")

	row := sqlClient.QueryRow("SELECT count(*) FROM sys.databases where name = ?", dbName)
	dbCount := 0
	row.Scan(&dbCount)
	if dbCount == 0 {
		t.Errorf("Database was not created")
	}
}
