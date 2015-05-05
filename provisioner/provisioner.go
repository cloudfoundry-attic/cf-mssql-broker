package provisioner

import (
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

// fmt template paramters: 1.databaseId
var createRoleWithFullDboAccessTemplate = []string{
	"use [%[1]v]",
	"create role [cf_bindings]",
	`grant	alter, control, create sequence,
	       	create sequence, delete, execute,
		   	insert, references, select, update,
		   	view change tracking, view definition, 
			take ownership
		on schema::[dbo] to [cf_bindings]
	`,
	"use master",
}

// fmt template parameters: 1.databaseId
var deleteDatabaseTemplate = []string{
	"use master",
	"alter database [%[1]v] set single_user with rollback immediate",
	"drop database [%[1]v]",
}

// fmt template parameters: 1.databaseId, 2.userId, 3.password, 4.dbRole
var createUserTemplate = []string{
	"use [%[1]v]",
	"create user [%[2]v] with password='%[3]v', default_schema=[dbo] ",
	"alter role [%[4]v] add member [%[2]v]",
	"alter role [cf_bindings] add member [%[2]v]",
	"use master",
}

// fmt template parameters: 1.databaseId, 2.userId
var deleteUserTemplate = []string{
	"use [%[1]v]",
	"drop user [%[2]v]",
	"use master",
}

// fmt template paramters: 1.databaseId
var isDatabaseCreatedTemplate = "select count(*)  from [master].sys.databases  where name = '%[1]v'"

// fmt template parameters: 1.databaseId, 2.userId
var isUserCreatedTemplate = "select count(*)  from [%[1]v].sys.database_principals  where name = '%[2]v'"

type MssqlProvisioner struct {
	dbClient         *sql.DB
	goSqlDriver      string
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

func NewMssqlProvisioner(logger lager.Logger, goSqlDriver string, connectionParams map[string]string) *MssqlProvisioner {
	return &MssqlProvisioner{
		dbClient:         nil,
		goSqlDriver:      goSqlDriver,
		connectionParams: connectionParams,
		logger:           logger,
	}
}

func (provisioner *MssqlProvisioner) Init() error {
	var err error = nil
	connString := buildConnectionString(provisioner.connectionParams)
	provisioner.dbClient, err = sql.Open(provisioner.goSqlDriver, connString)
	if err != nil {
		return err
	}

	err = provisioner.dbClient.Ping()
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *MssqlProvisioner) CreateDatabase(databaseId string) error {
	err := provisioner.executeTemplateWithoutTx(createDatabaseTemplate, databaseId)
	if err != nil {
		return err
	}

	err = provisioner.executeTemplateWithTx(createRoleWithFullDboAccessTemplate, databaseId)
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *MssqlProvisioner) DeleteDatabase(databaseId string) error {
	return provisioner.executeTemplateWithoutTx(deleteDatabaseTemplate, databaseId)
}

func (provisioner *MssqlProvisioner) CreateUser(databaseId, userId, password string, grantDbOwner bool) error {
	role := "db_ddladmin"
	if grantDbOwner {
		role = "db_owner"
	}
	return provisioner.executeTemplateWithoutTx(createUserTemplate, databaseId, userId, password, role)
}

func (provisioner *MssqlProvisioner) DeleteUser(databaseId, userId string) error {
	return provisioner.executeTemplateWithoutTx(deleteUserTemplate, databaseId, userId)
}

func (provisioner *MssqlProvisioner) IsDatabaseCreated(databaseId string) (bool, error) {
	res := 0

	err := provisioner.queryScalarTemplate(isDatabaseCreatedTemplate, &res, databaseId)
	if err != nil {
		return false, err
	}
	if res == 1 {
		return true, nil
	}
	return false, nil
}

func (provisioner *MssqlProvisioner) IsUserCreated(databaseId, userId string) (bool, error) {
	res := 0

	err := provisioner.queryScalarTemplate(isUserCreatedTemplate, &res, databaseId, userId)
	if err != nil {
		return false, err
	}
	if res == 1 {
		return true, nil
	}
	return false, nil
}

func (provisioner *MssqlProvisioner) queryScalarTemplate(template string, output interface{}, targs ...interface{}) error {
	sqlLine := compileTemplate(template, targs...)

	provisioner.logger.Debug("mssql-exec", lager.Data{"query": sqlLine})
	rowRes := provisioner.dbClient.QueryRow(sqlLine)

	err := rowRes.Scan(output)
	if err != nil {
		provisioner.logger.Error("mssql-exec", err, lager.Data{"query": sqlLine})
		return err
	}

	return nil
}

func (provisioner *MssqlProvisioner) executeTemplateWithTx(template []string, targs ...interface{}) error {
	tx, err := provisioner.dbClient.Begin()
	if err != nil {
		return err
	}

	for _, templateLine := range template {
		sqlLine := compileTemplate(templateLine, targs...)

		provisioner.logger.Debug("mssql-exec", lager.Data{"query": sqlLine})
		_, err = tx.Exec(sqlLine)
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				panic(rollbackErr.Error())
			}
			provisioner.logger.Error("mssql-exec", err, lager.Data{"query": sqlLine})
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (provisioner *MssqlProvisioner) executeTemplateWithoutTx(template []string, targs ...interface{}) error {
	for _, templateLine := range template {
		sqlLine := compileTemplate(templateLine, targs...)

		provisioner.logger.Debug("mssql-exec", lager.Data{"query": sqlLine})
		_, err := provisioner.dbClient.Exec(sqlLine)
		if err != nil {
			provisioner.logger.Error("mssql-exec", err, lager.Data{"query": sqlLine})
			return err
		}
	}

	return nil
}

func compileTemplate(template string, targs ...interface{}) string {
	compiled := fmt.Sprintf(template, targs...)
	extraErrorStart := strings.LastIndex(compiled, "%!(EXTRA")
	if extraErrorStart != -1 {
		// trim the extra args errs from sprintf
		compiled = compiled[0:extraErrorStart]
	}
	return compiled
}
