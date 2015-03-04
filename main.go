package main

import (
	"cf-mssql-broker/provisioner"
	"crypto/rand"
	"encoding/base64"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-golang/lager"
	"net/http"
	"os"
	"runtime"
)

type mssqlServiceBroker struct{}

type MssqlCredentials struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
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

var logger = lager.NewLogger("mssql-service-broker")
var mssqlProv *provisioner.MssqlProvisioner
var pars map[string]string

func (*mssqlServiceBroker) Services() []brokerapi.Service {
	// Return a []brokerapi.Service here, describing your service(s) and plan(s)
	logger.Debug("catalog-called")

	return []brokerapi.Service{
		brokerapi.Service{
			ID:          "b6844738-382b-4a9e-9f80-2ff5049d512f",
			Name:        "hp-mssql-dev",
			Description: "Microsoft SQL Server service for application development and testing",
			Bindable:    true,
			Plans: []brokerapi.ServicePlan{
				brokerapi.ServicePlan{
					ID:          "fb740fd7-2029-467a-9256-63ecd882f11c",
					Name:        "100m-dev",
					Description: "Shared SQL Server",
					Metadata: brokerapi.ServicePlanMetadata{
						Bullets:     []string{},
						DisplayName: "Mssql",
					},
				},
			},
		},
	}

}

func (*mssqlServiceBroker) Provision(instanceID string, serviceDetails brokerapi.ServiceDetails) error {
	// Provision a new instance here
	logger.Debug("provision-called", lager.Data{"instanceId": instanceID, "serviceDetails": serviceDetails})

	err := mssqlProv.CreateDatabase(instanceID)

	return err
}

func (*mssqlServiceBroker) Deprovision(instanceID string) error {
	// Deprovision instances here
	logger.Debug("deprovision-called", lager.Data{"instanceId": instanceID})

	err := mssqlProv.DeleteDatabase(instanceID)

	return err
}

func (*mssqlServiceBroker) Bind(instanceID, bindingID string) (interface{}, error) {
	// Bind to instances here
	// Return credentials which will be marshalled to JSON

	logger.Debug("bind-called", lager.Data{"instanceId": instanceID, "bindingId": bindingID})

	username := instanceID + "-" + bindingID
	password := randomString(32) + "qwerASF1234!@#$"

	err := mssqlProv.CreateBinding(instanceID, username, password)
	if err != nil {
		return nil, err
	}

	return MssqlCredentials{
		Host:     pars["server"],
		Port:     1433,
		Username: username,
		Password: password,
	}, nil
}

func (*mssqlServiceBroker) Unbind(instanceID, bindingID string) error {
	// Unbind from instances here
	logger.Debug("unbind-called", lager.Data{"instanceId": instanceID, "bindingId": bindingID})

	username := instanceID + "-" + bindingID
	err := mssqlProv.DelteBinding(instanceID, username)
	return err
}

func geListeningPort() string {
	res := os.Getenv("PORT") // CF and Heroku will set this env var
	if res == "" {
		res = "3000"

	}
	return res
}

func main() {
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	pars = map[string]string{
		"server":   "localhost\\sqlexpress",
		"database": "master",
	}

	msuser := "sa"
	mspass := "password1234!"
	if runtime.GOOS != "windows" {
		pars["driver"] = "freetds"
		pars["uid"] = msuser
		pars["pwd"] = mspass
		pars["port"] = "1433"
	} else {
		pars["driver"] = "sql server"
		if len(msuser) == 0 {
			pars["trusted_connection"] = "yes"
		} else {
			pars["uid"] = msuser
			pars["pwd"] = mspass
		}
	}

	mssqlProv = provisioner.NewMssqlProvisioner(logger, pars)
	mssqlProv.Init()

	serviceBroker := &mssqlServiceBroker{}

	credentials := brokerapi.BrokerCredentials{
		Username: "username",
		Password: "password",
	}

	brokerAPI := brokerapi.New(serviceBroker, logger, credentials)
	http.Handle("/", brokerAPI)

	addr := ":" + geListeningPort()
	logger.Debug("start-listening", lager.Data{"addr": addr})

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Error("error-listenting", err)
	}
}
