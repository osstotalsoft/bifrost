package config

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"os"
)

type Config struct {
	Endpoints              []Endpoint `json:"endpoints"`
	Port                   int        `json:"port"`
	Version                string     `json:"version"`
	UpstreamPathPrefix     string     `json:"upstream_path_prefix"`
	DownstreamPathPrefix   string     `json:"downstream_path_prefix"`
	LogLevel               string     `json:"log_level"`
	InCluster              bool       `json:"in_cluster"`
	OverrideServiceAddress string     `json:"override_service_address"`
}

type Endpoint struct {
	UpstreamPath         string   `json:"upstream_path"`
	UpstreamPathPrefix   string   `json:"upstream_path_prefix"`
	DownstreamPath       string   `json:"downstream_path"`
	DownstreamPathPrefix string   `json:"downstream_path_prefix"`
	ServiceName          string   `json:"service_name"`
	Methods              []string `json:"methods"`
	HandlerType          string   `json:"handler_type"`
	Topic                string   `json:"topic"`
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
