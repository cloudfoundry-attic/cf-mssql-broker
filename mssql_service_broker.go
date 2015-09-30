package main

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-golang/lager"
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

func secureRandomString(bytesOfEntpry int) string {
	rb := make([]byte, bytesOfEntpry)
	_, err := rand.Read(rb)

	if err != nil {
		logger.Fatal("rng-failure", err)
	}

	return base64.URLEncoding.EncodeToString(rb)
}

type mssqlServiceBroker struct{}

func (*mssqlServiceBroker) Services() []brokerapi.Service {
	// Return a []brokerapi.Service here, describing your service(s) and plan(s)
	logger.Info("catalog-called")

	return brokerConfig.ServiceCatalog
}

func (*mssqlServiceBroker) Provision(instanceID string, serviceDetails brokerapi.ProvisionDetails) error {
	// Provision a new instance here
	logger.Info("provision-called", lager.Data{"instanceId": instanceID, "serviceDetails": serviceDetails})

	databaseName := brokerConfig.DbIdentifierPrefix + instanceID

	exist, err := mssqlProv.IsDatabaseCreated(databaseName)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	if exist {
		return brokerapi.ErrInstanceAlreadyExists
	}

	err = mssqlProv.CreateDatabase(databaseName)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	return nil
}

func (*mssqlServiceBroker) Deprovision(instanceID string) error {
	// Deprovision instances here
	logger.Info("deprovision-called", lager.Data{"instanceId": instanceID})

	databaseName := brokerConfig.DbIdentifierPrefix + instanceID

	exist, err := mssqlProv.IsDatabaseCreated(databaseName)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	if !exist {
		return brokerapi.ErrInstanceDoesNotExist
	}

	err = mssqlProv.DeleteDatabase(databaseName)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	return nil
}

func (*mssqlServiceBroker) Bind(instanceID, bindingID string, bindDetails brokerapi.BindDetails) (interface{}, error) {
	// Bind to instances here
	// Return credentials which will be marshalled to JSON

	logger.Info("bind-called", lager.Data{"instanceId": instanceID, "bindingId": bindingID, "bindDetails": bindDetails})

	databaseName := brokerConfig.DbIdentifierPrefix + instanceID
	username := databaseName + "-" + bindingID
	password := secureRandomString(32) + happySqlPasswordPolicySuffix

	exist, err := mssqlProv.IsDatabaseCreated(databaseName)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	if !exist {
		return nil, brokerapi.ErrInstanceDoesNotExist
	}

	exist, err = mssqlProv.IsUserCreated(databaseName, username)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	if exist {
		return nil, brokerapi.ErrBindingAlreadyExists
	}

	err = mssqlProv.CreateUser(databaseName, username, password)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	bindingInfo := MssqlBindingCredentials{
		Hostname:         brokerConfig.ServedBindingHostname,
		Host:             brokerConfig.ServedBindingHostname,
		Port:             brokerConfig.ServedBindingPort,
		Name:             databaseName,
		Username:         username,
		Password:         password,
		ConnectionString: generateConnectionString(brokerConfig.ServedBindingHostname, brokerConfig.ServedBindingPort, databaseName, username, password),
	}

	return bindingInfo, nil
}

func (*mssqlServiceBroker) Unbind(instanceID, bindingID string) error {
	// Unbind from instances here
	logger.Info("unbind-called", lager.Data{"instanceId": instanceID, "bindingId": bindingID})

	databaseName := brokerConfig.DbIdentifierPrefix + instanceID
	username := databaseName + "-" + bindingID

	exist, err := mssqlProv.IsDatabaseCreated(databaseName)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	if !exist {
		return brokerapi.ErrInstanceDoesNotExist
	}

	exist, err = mssqlProv.IsUserCreated(databaseName, username)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	if !exist {
		return brokerapi.ErrBindingAlreadyExists
	}

	err = mssqlProv.DeleteUser(databaseName, username)
	if err != nil {
		logger.Fatal("provisioner-error", err)
	}

	return nil
}
