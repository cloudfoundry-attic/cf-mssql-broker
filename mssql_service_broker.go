package main

import (
	"crypto/rand"
	"encoding/base64"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-golang/lager"
)

func randomString(size int) string {
	rb := make([]byte, size)
	_, err := rand.Read(rb)

	if err != nil {
		logger.Fatal("rng-failure", err)
	}

	return base64.URLEncoding.EncodeToString(rb)
}

type mssqlServiceBroker struct{}

func (*mssqlServiceBroker) Services() []brokerapi.Service {
	// Return a []brokerapi.Service here, describing your service(s) and plan(s)
	logger.Debug("catalog-called")

	return brokerConfig.ServiceCatalog
}

func (*mssqlServiceBroker) Provision(instanceID string, serviceDetails brokerapi.ServiceDetails) error {
	// Provision a new instance here
	logger.Debug("provision-called", lager.Data{"instanceId": instanceID, "serviceDetails": serviceDetails})

	databaseName := brokerConfig.DbIdentifierPrefix + instanceID
	err := mssqlProv.CreateDatabase(databaseName)

	return err
}

func (*mssqlServiceBroker) Deprovision(instanceID string) error {
	// Deprovision instances here
	logger.Debug("deprovision-called", lager.Data{"instanceId": instanceID})

	databaseName := brokerConfig.DbIdentifierPrefix + instanceID
	err := mssqlProv.DeleteDatabase(databaseName)

	return err
}

func (*mssqlServiceBroker) Bind(instanceID, bindingID string) (interface{}, error) {
	// Bind to instances here
	// Return credentials which will be marshalled to JSON

	logger.Debug("bind-called", lager.Data{"instanceId": instanceID, "bindingId": bindingID})

	databaseName := brokerConfig.DbIdentifierPrefix + instanceID
	username := databaseName + "-" + bindingID
	password := randomString(32) + "qwerASF1234!@#$"

	err := mssqlProv.CreateUser(databaseName, username, password)
	if err != nil {
		return nil, err
	}

	return MssqlBindingCredentials{
		Hostname: brokerConfig.ServedBindingHostname,
		Port:     brokerConfig.ServedBindingPort,
		Name:     databaseName,
		Username: username,
		Password: password,
	}, nil
}

func (*mssqlServiceBroker) Unbind(instanceID, bindingID string) error {
	// Unbind from instances here
	logger.Debug("unbind-called", lager.Data{"instanceId": instanceID, "bindingId": bindingID})

	databaseName := brokerConfig.DbIdentifierPrefix + instanceID
	username := databaseName + "-" + bindingID
	err := mssqlProv.DeleteUser(databaseName, username)
	return err
}
