package broker

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-golang/lager"
	"github.com/cloudfoundry-incubator/cf-mssql-broker/config"
	"github.com/cloudfoundry-incubator/cf-mssql-broker/provisioner"
)

// This is a suffix added to the password that will prevent the default
// sql password policy to reject high entory passwords from a base64 char set.
// The purpose of this is to prenvent the admin from disableing the policy,
// thus it will simplify the inital setup, and it will NOT in any way reduce
// the final password strength.
// Policy descrpition:
// https://msdn.microsoft.com/en-us/library/ms161959.aspx
// http://xkcd.com/936/
const happySqlPasswordPolicySuffix = "Aa_0"

func secureRandomString(bytesOfEntpry int) (string, error) {
	rb := make([]byte, bytesOfEntpry)
	_, err := rand.Read(rb)

	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(rb), nil
}

type MssqlServiceBroker struct{
	brokerConfig *config.Config
	mssqlProv    *provisioner.MssqlProvisioner
	logger       lager.Logger
}

func NewMssqlServiceBroker(logger lager.Logger, config *config.Config, provisioner *provisioner.MssqlProvisioner) *MssqlServiceBroker{
	return &MssqlServiceBroker{
		brokerConfig: config,
		mssqlProv: 	  provisioner,
		logger:       logger,
	}
}

func (b *MssqlServiceBroker) Services() []brokerapi.Service {
	// Return a []brokerapi.Service here, describing your service(s) and plan(s)
	b.logger.Info("catalog-called")

	return b.brokerConfig.ServiceCatalog
}

func (b *MssqlServiceBroker) Provision(instanceID string, serviceDetails brokerapi.ProvisionDetails) error {
	// Provision a new instance here
	b.logger.Info("provision-called", lager.Data{"instanceId": instanceID, "serviceDetails": serviceDetails})

	databaseName := b.brokerConfig.DbIdentifierPrefix + instanceID

	exist, err := b.mssqlProv.IsDatabaseCreated(databaseName)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	if exist {
		return brokerapi.ErrInstanceAlreadyExists
	}

	err = b.mssqlProv.CreateDatabase(databaseName)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	return nil
}

func (b *MssqlServiceBroker) Deprovision(instanceID string) error {
	// Deprovision instances here
	b.logger.Info("deprovision-called", lager.Data{"instanceId": instanceID})

	databaseName := b.brokerConfig.DbIdentifierPrefix + instanceID

	exist, err := b.mssqlProv.IsDatabaseCreated(databaseName)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	if !exist {
		return brokerapi.ErrInstanceDoesNotExist
	}

	err = b.mssqlProv.DeleteDatabase(databaseName)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	return nil
}

func (b *MssqlServiceBroker) Bind(instanceID, bindingID string, bindDetails brokerapi.BindDetails) (interface{}, error) {
	// Bind to instances here
	// Return credentials which will be marshalled to JSON

	b.logger.Info("bind-called", lager.Data{"instanceId": instanceID, "bindingId": bindingID, "bindDetails": bindDetails})

	databaseName := b.brokerConfig.DbIdentifierPrefix + instanceID
	username := databaseName + "-" + bindingID
	
	randomString, err := secureRandomString(32)
	if err != nil {
		b.logger.Fatal("rng-failure", err)
	}
	
	password := randomString + happySqlPasswordPolicySuffix

	exist, err := b.mssqlProv.IsDatabaseCreated(databaseName)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	if !exist {
		return nil, brokerapi.ErrInstanceDoesNotExist
	}

	exist, err = b.mssqlProv.IsUserCreated(databaseName, username)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	if exist {
		return nil, brokerapi.ErrBindingAlreadyExists
	}

	err = b.mssqlProv.CreateUser(databaseName, username, password)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	bindingInfo := MssqlBindingCredentials{
		Hostname:         b.brokerConfig.ServedBindingHostname,
		Host:             b.brokerConfig.ServedBindingHostname,
		Port:             b.brokerConfig.ServedBindingPort,
		Name:             databaseName,
		Username:         username,
		Password:         password,
		ConnectionString: generateConnectionString(b.brokerConfig.ServedBindingHostname, b.brokerConfig.ServedBindingPort, databaseName, username, password),
	}

	return bindingInfo, nil
}

func (b *MssqlServiceBroker) Unbind(instanceID, bindingID string) error {
	// Unbind from instances here
	b.logger.Info("unbind-called", lager.Data{"instanceId": instanceID, "bindingId": bindingID})

	databaseName := b.brokerConfig.DbIdentifierPrefix + instanceID
	username := databaseName + "-" + bindingID

	exist, err := b.mssqlProv.IsDatabaseCreated(databaseName)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	if !exist {
		return brokerapi.ErrInstanceDoesNotExist
	}

	exist, err = b.mssqlProv.IsUserCreated(databaseName, username)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	if !exist {
		return brokerapi.ErrBindingAlreadyExists
	}

	err = b.mssqlProv.DeleteUser(databaseName, username)
	if err != nil {
		b.logger.Fatal("provisioner-error", err)
	}

	return nil
}
