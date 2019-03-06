package config

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"os"
)

type Config struct {
	Endpoints              []Endpoint `mapstructure:"endpoints"`
	Port                   int        `mapstructure:"port"`
	Version                string     `mapstructure:"version"`
	Name                   string     `mapstructure:"name"`
	UpstreamPathPrefix     string     `mapstructure:"upstream_path_prefix"`
	DownstreamPathPrefix   string     `mapstructure:"downstream_path_prefix"`
	LogLevel               string     `mapstructure:"log_level"`
	InCluster              bool       `mapstructure:"in_cluster"`
	OverrideServiceAddress string     `mapstructure:"override_service_address"`
}

type Endpoint struct {
	UpstreamPath         string                 `mapstructure:"upstream_path"`
	UpstreamPathPrefix   string                 `mapstructure:"upstream_path_prefix"`
	DownstreamPath       string                 `mapstructure:"downstream_path"`
	DownstreamPathPrefix string                 `mapstructure:"downstream_path_prefix"`
	ServiceName          string                 `mapstructure:"service_name"`
	Methods              []string               `mapstructure:"methods"`
	HandlerType          string                 `mapstructure:"handler_type"`
	HandlerConfig        map[string]interface{} `mapstructure:"handler_config"`
	Filters              map[string]interface{} `mapstructure:"filters"`
}

func LoadConfig() *Config {
	var config = new(Config)
	configFile, err := os.Open("config.json")
	defer configFile.Close()
	if err != nil {
		log.Fatalln(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(config)
	if err != nil {
		log.Fatalln(err.Error())
	}

	cfStrt, _ := json.Marshal(config)
	log.Infof("Using configuration : %s", string(cfStrt))
	return config
}
