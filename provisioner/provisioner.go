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
	tx, err := provisioner.dbClient.Begin()
	if err != nil {
		return err
	}

	sqlquery := "use master"
	provisioner.logger.Debug("mssql-query-create-database", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	sqlquery = "create database [" + id + "] containment = partial"
	provisioner.logger.Debug("mssql-query-create-database", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *MssqlProvisioner) DeleteDatabase(id string) error {
	tx, err := provisioner.dbClient.Begin()
	if err != nil {
		return err
	}

	sqlquery := "use master"
	provisioner.logger.Debug("mssql-query-delete-database", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	sqlquery = "alter database [" + id + "] set single_user with rollback immediate"
	provisioner.logger.Debug("mssql-query-delete-database", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	sqlquery = "drop database [" + id + "]"
	provisioner.logger.Debug("mssql-query-delete-database", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *MssqlProvisioner) CreateUser(dbId, userId, password string) error {
	tx, err := provisioner.dbClient.Begin()
	if err != nil {
		return err
	}

	sqlquery := "use [" + dbId + "]"
	provisioner.logger.Debug("mssql-query-create-user", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	sqlquery = "create user [" + userId + "] with password='" + password + "'"
	provisioner.logger.Debug("mssql-query-create-user", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	sqlquery = "alter role  [db_owner] add member [" + userId + "]"
	provisioner.logger.Debug("mssql-query-create-user", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *MssqlProvisioner) DeleteUser(dbId, userId string) error {
	tx, err := provisioner.dbClient.Begin()
	if err != nil {
		return err
	}

	sqlquery := "use [" + dbId + "]"
	provisioner.logger.Debug("mssql-query-delete-user", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	sqlquery = "drop user [" + userId + "]"
	provisioner.logger.Debug("mssql-query-delete-user", lager.Data{"query": sqlquery})
	_, err = tx.Exec(sqlquery)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
