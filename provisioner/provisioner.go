package provisioner

import (
	_ "code.google.com/p/odbc"
	"database/sql"
	"github.com/pivotal-golang/lager"
)

type MssqlProvisioner struct {
	dbClient         *sql.DB
	connectionParams map[string]string
	logger           lager.Logger
}

func buildConnectionString(connectionParams map[string]string) string {
	var res string = ""
	for k, v := range connectionParams {
		res += k + "=" + v + ";"
	}
	return res
}

func NewMssqlProvisioner(logger lager.Logger, connectionParams map[string]string) *MssqlProvisioner {
	return &MssqlProvisioner{
		dbClient:         nil,
		connectionParams: connectionParams,
		logger:           logger,
	}
}

func (provisioner *MssqlProvisioner) Init() error {
	var err error = nil
	connString := buildConnectionString(provisioner.connectionParams)
	provisioner.dbClient, err = sql.Open("odbc", connString)
	provisioner.dbClient.SetMaxOpenConns(1)

	return err
}

func (provisioner *MssqlProvisioner) CreateDatabase(id string) error {
	sqlquery := "use master; create database [" + id + "] containment = partial"
	provisioner.logger.Debug("mssql-query-create-database", lager.Data{"query": sqlquery})
	_, err := provisioner.dbClient.Exec(sqlquery)

	return err
}

func (provisioner *MssqlProvisioner) DeleteDatabase(id string) error {
	// _, err := provisioner.dbClient.Exec("drop database " + id)
	sqlquery := "use master; alter database [" + id + "] set single_user with rollback immediate; drop database [" + id + "]"
	provisioner.logger.Debug("mssql-query-delete-database", lager.Data{"query": sqlquery})
	_, err := provisioner.dbClient.Exec(sqlquery)

	return err
}

func (provisioner *MssqlProvisioner) CreateUser(dbId, userId, password string) error {
	tx, err := provisioner.dbClient.Begin()
	if err != nil {
		return err
	}

	sqlquery := "use [" + dbId + "]; create user [" + userId + "] with password='" + password + "' ; use master "
	provisioner.logger.Debug("mssql-query-create-user", lager.Data{"query": sqlquery})
	tx.Exec(sqlquery)

	sqlquery = "use [" + dbId + "]; alter role  [db_owner] add member [" + dbId + "] ; use master "
	provisioner.logger.Debug("mssql-query-create-user", lager.Data{"query": sqlquery})
	tx.Exec(sqlquery)

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *MssqlProvisioner) DeleteUser(dbId, userId string) error {
	sqlquery := "use [" + dbId + "]; drop user [" + userId + "] ; use master "
	provisioner.logger.Debug("mssql-query-delete-user", lager.Data{"query": sqlquery})
	_, err := provisioner.dbClient.Exec(sqlquery)

	return err
}

func (provisioner *MssqlProvisioner) DeleteBinding(dbId, userId string) error {
        _, err := provisioner.dbClient.Exec("use [" + dbId + "]; drop user " + userId)
        return err
}

func (provisioner *MssqlProvisioner) Deprovision(dbId string) error {

        _, _ := provisioner.dbClient.Exec("ALTER DATABASE [" + dbId + "] SET OFFLINE WITH ROLLBACK IMMEDIATE")
        _, err := provisioner.dbClient.Exec("drop database [" + dbId + "]")
        return err
}
