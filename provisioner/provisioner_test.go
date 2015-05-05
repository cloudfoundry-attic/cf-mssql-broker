package provisioner

import (
	"database/sql"
	"fmt"
	"github.com/pivotal-golang/lager/lagertest"
	"testing"
)

var logger *lagertest.TestLogger = lagertest.NewTestLogger("mssql-provisioner")
var mssqlPars = map[string]string{
	"driver":             "sql server",
	"server":             "127.0.0.1",
	"database":           "master",
	"trusted_connection": "yes",
}

func TestCreateDatabaseOdbcDriver(t *testing.T) {
	dbName := "cf-broker-testing.create-db"

	sqlClient, err := sql.Open("odbc", buildConnectionString(mssqlPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", mssqlPars)
	err = mssqlProv.Init()
	if err != nil {
		t.Errorf("Provisioner init error, %v", err)
	}

	// Act
	err = mssqlProv.CreateDatabase(dbName)

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

func TestDeleteDatabaseOdbcDriver(t *testing.T) {
	dbName := "cf-broker-testing.delete-db"

	sqlClient, err := sql.Open("odbc", buildConnectionString(mssqlPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", mssqlPars)
	err = mssqlProv.Init()
	if err != nil {
		t.Errorf("Database init error, %v", err)
	}
	err = mssqlProv.CreateDatabase(dbName)

	// Act

	err = mssqlProv.DeleteDatabase(dbName)

	// Assert
	if err != nil {
		t.Errorf("Database delete error, %v", err)
	}

	row := sqlClient.QueryRow("SELECT count(*) FROM sys.databases where name = ?", dbName)
	dbCount := 0
	row.Scan(&dbCount)
	if dbCount != 0 {
		t.Errorf("Database %s was not deleted", dbName)
	}
}

func TestDeleteUserOdbcDriver(t *testing.T) {
	dbName := "cf-broker-testing.create-db"
	userNanme := "cf-broker-testing.create-user"

	sqlClient, err := sql.Open("odbc", buildConnectionString(mssqlPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", mssqlPars)
	err = mssqlProv.Init()
	if err != nil {
		t.Errorf("Provisioner init error, %v", err)
	}

	err = mssqlProv.CreateDatabase(dbName)
	if err != nil {
		t.Errorf("Database create error, %v", err)
	}

	// Act
	err = mssqlProv.CreateUser(dbName, userNanme, "passwordAa_0", true)

	// Assert
	if err != nil {
		t.Errorf("User create error, %v", err)
	}
	defer sqlClient.Exec("drop database [" + dbName + "]")

	row := sqlClient.QueryRow(fmt.Sprintf("select count(*)  from [%s].sys.database_principals  where name = ?", dbName), userNanme)
	dbCount := 0
	row.Scan(&dbCount)
	if dbCount == 0 {
		t.Errorf("User was not created")
	}
}

func TestCreateUserOdbcDriver(t *testing.T) {
	dbName := "cf-broker-testing.create-db"
	userNanme := "cf-broker-testing.create-user"

	sqlClient, err := sql.Open("odbc", buildConnectionString(mssqlPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", mssqlPars)
	err = mssqlProv.Init()
	if err != nil {
		t.Errorf("Provisioner init error, %v", err)
	}

	err = mssqlProv.CreateDatabase(dbName)
	if err != nil {
		t.Errorf("Database create error, %v", err)
	}
	err = mssqlProv.CreateUser(dbName, userNanme, "passwordAa_0", false)
	if err != nil {
		t.Errorf("User create error, %v", err)
	}

	// Act
	exists, err := mssqlProv.IsUserCreated(dbName, userNanme)

	// Assert
	if err != nil {
		t.Errorf("IsUserCreated error, %v", err)
	}
	if !exists {
		t.Errorf("IsUserCreated returned false, expected true")
	}

	// Act
	err = mssqlProv.DeleteUser(dbName, userNanme)

	// Assert
	if err != nil {
		t.Errorf("User delete error, %v", err)
	}

	// Act
	exists, err = mssqlProv.IsUserCreated(dbName, userNanme)

	// Assert
	if err != nil {
		t.Errorf("IsUserCreated error, %v", err)
	}
	if exists {
		t.Errorf("IsUserCreated returned true, expected false")
	}

	defer sqlClient.Exec("drop database [" + dbName + "]")

	row := sqlClient.QueryRow(fmt.Sprintf("select count(*)  from [%s].sys.database_principals  where name = ?", dbName), userNanme)
	dbCount := 0
	row.Scan(&dbCount)
	if dbCount != 0 {
		t.Errorf("User was not deleted")
	}
}

func TestIsDatabaseCreatedOdbcDriver(t *testing.T) {
	dbName := "cf-broker-testing.nonexisting-db"

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", mssqlPars)
	err := mssqlProv.Init()
	if err != nil {
		t.Errorf("Provisioner init error, %v", err)
	}

	// Act
	exists, err := mssqlProv.IsDatabaseCreated(dbName)

	// Assert
	if err != nil {
		t.Errorf("Check for database error, %v", err)
	}
	if exists {
		t.Errorf("Check for database error, expected false, but received true")
	}
}

func TestIsDatabaseCreatedOdbcDriver2(t *testing.T) {
	dbName := "cf-broker-testing.create-db"

	sqlClient, err := sql.Open("odbc", buildConnectionString(mssqlPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", mssqlPars)
	mssqlProv.Init()
	if err != nil {
		t.Errorf("Provisioner init error, %v", err)
	}
	err = mssqlProv.CreateDatabase(dbName)
	if err != nil {
		t.Errorf("Database create error, %v", err)
	}

	// Act
	exists, err := mssqlProv.IsDatabaseCreated(dbName)

	// Assert
	if err != nil {
		t.Errorf("Check for database error, %v", err)
	}
	if !exists {
		t.Errorf("Check for database error, expected true, but received false")
	}

	defer sqlClient.Exec("drop database [" + dbName + "]")
}
