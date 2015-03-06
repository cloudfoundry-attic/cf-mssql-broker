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
	}l
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

func (provisioner *MssqlProvisioner) CreateDatabase(id string) error {
	_, err := provisioner.dbClient.Exec("create database " + id + " containment = partial")
	return err
}

func (provisioner *MssqlProvisioner) DeleteDatabase(id string) error {
	// _, err := provisioner.dbClient.Exec("drop database " + id)
	_, err := provisioner.dbClient.Exec("alter database [" + id + "] set single_user with rollback immediate; drop database " + id)
	return err
}

func (provisioner *MssqlProvisioner) CreateBinding(dbId, userId, password string) error {
	tx, err := provisioner.dbClient.Begin()
	if err != nil {
		return err
	}

	tx.Exec("use [" + dbId + "]; create user [" + userId + "] with password='" + password + "' ") 

	tx.Exec("use [" + dbId + "]; alter role  [db_owner] add member [" + dbId + "] ")

	// alt for making a user db_owner
	// tx.Exec("use [" + dbId + "]; exec sp_addrolemember N'db_owner', N'" + dbId + "' ")

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
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
