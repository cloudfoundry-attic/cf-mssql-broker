package provisioner

import (
	_ "code.google.com/p/odbc"
	"database/sql"
	"fmt"
	"github.com/pivotal-golang/lager"
	"strings"
)

// Sql templates are executed as a trascation and on query per array element
// The templates can be extracted to an external file (e.g. json or yaml)

// fmt template paramters: 1.databaseId
var createDatabaseTemplate = []string{
	"use master",
	"create database [%[1]v] containment = partial",
}

// fmt template parameters: 1.databaseId
var deleteDatabaseTemplate = []string{
	"use master",
	"alter database [%[1]v] set single_user with rollback immediate",
	"drop database [%[1]v]",
}

// fmt template parameters: 1.databaseId, 2.userId, 3.password
var createUserTemplate = []string{
	"use [%[1]v]",
	"create user [%[2]v] with password='%[3]v'",
	"alter role  [db_owner] add member [%[2]v]",
}

// fmt template parameters: 1.databaseId, 2.userId
var deleteUserTemplate = []string{
	"use [%[1]v]",
	"drop user [%[2]v]",
}

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

	return err
}

func (provisioner *MssqlProvisioner) CreateDatabase(databaseId string) error {
	return provisioner.executeTemplate(createDatabaseTemplate, databaseId)
}

func (provisioner *MssqlProvisioner) DeleteDatabase(databaseId string) error {
	return provisioner.executeTemplate(deleteDatabaseTemplate, databaseId)
}

func (provisioner *MssqlProvisioner) CreateUser(databaseId, userId, password string) error {
	return provisioner.executeTemplate(createUserTemplate, databaseId, userId, password)
}

func (provisioner *MssqlProvisioner) DeleteUser(databaseId, userId string) error {
	return provisioner.executeTemplate(deleteUserTemplate, databaseId, userId)
}

func (provisioner *MssqlProvisioner) executeTemplate(template []string, targs ...interface{}) error {
	tx, err := provisioner.dbClient.Begin()
	if err != nil {
		return err
	}

	for _, templateLine := range template {
		// more details or alternatives here:

		sqlLine := fmt.Sprintf(templateLine, targs...)
		extraErrorStart := strings.LastIndex(sqlLine, "%!(EXTRA")
		if extraErrorStart != -1 {
			// trim the extra args errs from sprintf
			sqlLine = sqlLine[0:extraErrorStart]
		}

		provisioner.logger.Debug("mssql-exec", lager.Data{"query": sqlLine})
		_, err = tx.Exec(sqlLine)
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				panic(rollbackErr.Error())
			}
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
