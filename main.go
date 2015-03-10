package main

import (
	"cf-mssql-broker/config"
	"cf-mssql-broker/provisioner"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-golang/lager"
	"net/http"
	"os"
	"runtime"
)

type mssqlServiceBroker struct{}

type MssqlCredentials struct {
	Hostname string `json:"hostname"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func randomString(size int) string {
	rb := make([]byte, size)
	_, err := rand.Read(rb)

	if err != nil {
		logger.Fatal("rng-failure", err)
	}

	return base64.URLEncoding.EncodeToString(rb)
}

var configFile = flag.String("config", "", "Location of the Mssql Service Broker config json file")
var brokerConfig *config.Config

var logger = lager.NewLogger("mssql-service-broker")
var mssqlProv *provisioner.MssqlProvisioner
var pars map[string]string

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

	return MssqlCredentials{
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

func getListeningAddr(config *config.Config) string {
	// CF and Heroku will set this env var for their hosted apps
	envPort := os.Getenv("PORT")
	if envPort == "" {
		if len(config.ListeningAddr) == 0 {
			return ":3000"
		}
		return config.ListeningAddr
	}

	return ":" + envPort
}

func main() {
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	flag.Parse()
	var err error
	brokerConfig, err = config.LoadFromFile(*configFile)
	if err != nil {
		logger.Fatal("config-load-error", err)
	}

	logger.Debug("config-file", lager.Data{"config": brokerConfig})

	mssqlPars := brokerConfig.MssqlOdbcConnection

	// set default sql driver if it is not set based on the OS
	if _, ok := mssqlPars["driver"]; !ok {
		if runtime.GOOS != "windows" {
			pars["driver"] = "freetds"
		} else {
			pars["driver"] = "sql server"
		}
	}

	mssqlProv = provisioner.NewMssqlProvisioner(logger, mssqlPars)
	mssqlProv.Init()

	serviceBroker := &mssqlServiceBroker{}

	brokerAPI := brokerapi.New(serviceBroker, logger, brokerConfig.Crednetials)
	http.Handle("/", brokerAPI)

	addr := getListeningAddr(brokerConfig)
	logger.Debug("start-listening", lager.Data{"addr": addr})

	err = http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Error("error-listenting", err)
	}
}
