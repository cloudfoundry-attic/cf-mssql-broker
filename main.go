package main

import (
	"flag"
	"fmt"
	"github.com/hpcloud/cf-mssql-broker/config"
	"github.com/hpcloud/cf-mssql-broker/provisioner"
	"github.com/pivotal-cf/brokerapi"
	"github.com/pivotal-golang/lager"
	"net/http"
	"os"
	"runtime"
)

const (
	DEBUG = "debug"
	INFO  = "info"
	ERROR = "error"
	FATAL = "fatal"
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

func getLogLevel(config *config.Config) lager.LogLevel {
	var minLogLevel lager.LogLevel
	switch config.LogLevel {
	case DEBUG:
		minLogLevel = lager.DEBUG
	case INFO:
		minLogLevel = lager.INFO
	case ERROR:
		minLogLevel = lager.ERROR
	case FATAL:
		minLogLevel = lager.FATAL
	default:
		panic(fmt.Errorf("invalid log level: %s", config.LogLevel))
	}

	return minLogLevel
}

func runMain() {

	if !flag.Parsed() {
		flag.Parse()
	}
	var err error
	brokerConfig, err = config.LoadFromFile(*configFile)

	if err != nil {
		panic(fmt.Errorf("configuration load error from file %s. Err: %s", *configFile, err))
	}

	logger.RegisterSink(lager.NewWriterSink(os.Stdout, getLogLevel(brokerConfig)))

	logger.Debug("config-load-success", lager.Data{"file-source": *configFile, "config": brokerConfig})

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
	err = mssqlProv.Init()
	if err != nil {
		logger.Fatal("error-initializing-provisioner", err)
	}

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
