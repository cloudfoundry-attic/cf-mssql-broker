package config

import (
	"encoding/json"
	"github.com/pivotal-cf/brokerapi"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Config struct {
	DbIdentifierPrefix    string                      `json:"dbIdentifierPrefix"`
	ListeningAddr         string                      `json:"listeningAddr"`
	Crednetials           brokerapi.BrokerCredentials `json:"brokerCredentials"`
	ServiceCatalog        []brokerapi.Service         `json:"serviceCatalog"`
	BrokerGoSqlDriver     string                      `json:"brokerGoSqlDriver"`
	BrokerMssqlConnection map[string]string           `json:"brokerMssqlConnection"`
	ServedBindingHostname string                      `json:"servedMssqlBindingHostname"`
	ServedBindingPort     int                         `json:"servedMssqlBindingPort"`
}

func LoadFromFile(path string) (*Config, error) {
	if len(path) == 0 {
		binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			return nil, err
		}
		path = filepath.Join(binDir, "cf_mssql_broker_config.json")
	}
	jsonConf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseJson(jsonConf)
}

func ParseJson(jsonConf []byte) (*Config, error) {
	config := &Config{}

	err := json.Unmarshal(jsonConf, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
