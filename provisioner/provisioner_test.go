package provisioner

import (
	"database/sql"
	"fmt"
	"github.com/pivotal-golang/lager/lagertest"
	"testing"
)

var logger *lagertest.TestLogger = lagertest.NewTestLogger("mssql-provisioner")

var odbcPars = map[string]string{
	"driver":             "{SQL Server Native Client 11.0}", // or with an older driver version "{SQL Server}"
	"server":             "127.0.0.1",                       // or (local)\\sqlexpress
	"database":           "master",
	"trusted_connection": "yes",
}

var mssqlPars = map[string]string{
	"server":   "127.0.0.1",
	"port":     "38017",
	"database": "master",
	"user id":  "sa",
	"password": "password",
}

func TestCreateDatabaseOdbcDriver(t *testing.T) {
	dbName := "cf-broker-testing.create-db"

	sqlClient, err := sql.Open("odbc", buildConnectionString(odbcPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", odbcPars)
	err = mssqlProv.Init()
	if err != nil {
		t.Errorf("Provisioner init error, %v", err)
	}
	defer mssqlProv.Close()

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

	sqlClient, err := sql.Open("odbc", buildConnectionString(odbcPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", odbcPars)
	err = mssqlProv.Init()
	if err != nil {
		t.Errorf("Database init error, %v", err)
	}
	defer mssqlProv.Close()

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

func TestCreateUserOdbcDriver(t *testing.T) {
	dbName := "cf-broker-testing.create-db"
	userNanme := "cf-broker-testing.create-user"

	sqlClient, err := sql.Open("odbc", buildConnectionString(odbcPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", odbcPars)
	err = mssqlProv.Init()

	if err != nil {
		t.Errorf("Provisioner init error, %v", err)
	}

	err = mssqlProv.CreateDatabase(dbName)
	if err != nil {
		t.Errorf("Database create error, %v", err)
	}

	// Act
	err = mssqlProv.CreateUser(dbName, userNanme, "passwordAa_0")

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

func TestDeleteUserOdbcDriver(t *testing.T) {
	dbName := "cf-broker-testing.create-db"
	userNanme := "cf-broker-testing.create-user"

	sqlClient, err := sql.Open("odbc", buildConnectionString(odbcPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", odbcPars)
	err = mssqlProv.Init()
	if err != nil {
		t.Errorf("Provisioner init error, %v", err)
	}
	defer mssqlProv.Close()

	err = mssqlProv.CreateDatabase(dbName)
	if err != nil {
		t.Errorf("Database create error, %v", err)
	}
	err = mssqlProv.CreateUser(dbName, userNanme, "passwordAa_0")
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
	mssqlProv := NewMssqlProvisioner(logger, "odbc", odbcPars)
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

	sqlClient, err := sql.Open("odbc", buildConnectionString(odbcPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", odbcPars)
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

func TestStressOdbcDriver(t *testing.T) {
	dbName := "cf-broker-testing.create-db"
	dbNameA := "cf-broker-testing.create-db-A"
	dbName2 := "cf-broker-testing.create-db-2"
	userNanme := "cf-broker-testing.create-user"

	sqlClient, err := sql.Open("odbc", buildConnectionString(odbcPars))
	defer sqlClient.Close()

	sqlClient.Exec("drop database [" + dbName + "]")
	sqlClient.Exec("drop database [" + dbNameA + "]")
	sqlClient.Exec("drop database [" + dbName2 + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "odbc", odbcPars)
	err = mssqlProv.Init()
	if err != nil {
		t.Errorf("Provisioner init error, %v", err)
	}

	err = mssqlProv.CreateDatabase(dbName)
	if err != nil {
		t.Errorf("Database create error, %v", err)
	}

	err = mssqlProv.CreateDatabase(dbNameA)
	if err != nil {
		t.Errorf("Database create error, %v", err)
	}

	wait := make(chan bool)

	go func() {
		for i := 1; i < 8; i++ {

			err := mssqlProv.CreateDatabase(dbName2)
			if err != nil {
				t.Errorf("Database create error, %v", err)
				break
			}

			err = mssqlProv.DeleteDatabase(dbName2)
			if err != nil {
				t.Errorf("Database delete error, %v", err)
				break
			}
		}

		wait <- true
	}()

	go func() {
		for i := 1; i < 32; i++ {
			err = mssqlProv.CreateUser(dbName, userNanme, "passwordAa_0")
			if err != nil {
				t.Errorf("User create error, %v", err)
				break
			}

			err = mssqlProv.DeleteUser(dbName, userNanme)
			if err != nil {
				t.Errorf("User delete error, %v", err)
				break
			}

		}

		wait <- true
	}()

	go func() {
		for i := 1; i < 32; i++ {
			err = mssqlProv.CreateUser(dbNameA, userNanme, "passwordAa_0")
			if err != nil {
				t.Errorf("User create error, %v", err)
				break
			}

			err = mssqlProv.DeleteUser(dbNameA, userNanme)
			if err != nil {
				t.Errorf("User delete error, %v", err)
				break
			}

		}

		wait <- true
	}()

	<-wait
	<-wait
	<-wait

	sqlClient.Exec("drop database [" + dbName + "]")
	sqlClient.Exec("drop database [" + dbName2 + "]")
	sqlClient.Exec("drop database [" + dbNameA + "]")
}

func TestCreateDatabaseMssqlDriver(t *testing.T) {
	dbName := "cf-broker-testing.create-db"

	sqlClient, err := sql.Open("mssql", buildConnectionString(mssqlPars))
	defer sqlClient.Close()

	err = sqlClient.Ping()
	if err != nil {
		t.Skipf("Could not connect with pure mssql driver to %v", mssqlPars)
		return
	}

	sqlClient.Exec("drop database [" + dbName + "]")

	logger = lagertest.NewTestLogger("process-controller")
	mssqlProv := NewMssqlProvisioner(logger, "mssql", mssqlPars)
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
