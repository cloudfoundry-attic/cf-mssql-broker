package main

import (
	"flag"
	"github.com/hpcloud/cf-mssql-broker/config"
	"github.com/hpcloud/cf-mssql-broker/provisioner"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-golang/lager"
	"net/http"
	"os"
	"runtime"
)

var configFile = flag.String("config", "", "Location of the Mssql Service Broker config json file")
var brokerConfig *config.Config

var logger = lager.NewLogger("mssql-service-broker")
var mssqlProv *provisioner.MssqlProvisioner

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
	logger.Info("config-load-successful", lager.Data{"file": configFile})

	logger.Debug("config-file", lager.Data{"config": brokerConfig})

	mssqlPars := brokerConfig.BrokerMssqlConnection

	// set default sql driver if it is not set based on the OS
	if _, ok := mssqlPars["driver"]; !ok && brokerConfig.BrokerGoSqlDriver == "odbc" {
		if runtime.GOOS != "windows" {
			mssqlPars["driver"] = "freetds"
		} else {
			mssqlPars["driver"] = "sql server"
		}
	}

	mssqlProv = provisioner.NewMssqlProvisioner(logger, brokerConfig.BrokerGoSqlDriver, mssqlPars)
	mssqlProv.Init()

	serviceBroker := &mssqlServiceBroker{}

	brokerAPI := brokerapi.New(serviceBroker, logger, brokerConfig.Crednetials)
	http.Handle("/", brokerAPI)

	addr := getListeningAddr(brokerConfig)
	logger.Info("start-listening", lager.Data{"addr": addr})

	err = http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Error("error-listenting", err)
	}
}
