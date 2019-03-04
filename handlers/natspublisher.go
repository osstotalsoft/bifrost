package handlers

import (
	"github.com/mitchellh/mapstructure"
	"github.com/osstotalsoft/bifrost/gateway"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type NatsConfig struct {
	NatsUrl     string `mapstructure:"nats_url"`
	Cluster     string `mapstructure:"cluster"`
	ClientId    string `mapstructure:"client_id"`
	QGroup      string `mapstructure:"q_group"`
	DurableName string `mapstructure:"durable_name"`
}

type NatsEndpointConfig struct {
	Topic string `mapstructure:"topic"`
}

func NewNatsPublisher(config NatsConfig) gateway.HandlerFunc {

	return func(endpoint gateway.Endpoint) http.Handler {
		var h http.Handler
		var cfg NatsEndpointConfig

		_ = mapstructure.Decode(endpoint.HandlerConfig, &cfg)

		h = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			if err := publish(config.Cluster, cfg.Topic, "", request); err != nil {
				http.Error(writer, err.Error(), 500)
				return
			}

			writer.WriteHeader(200)
		})

		return h
	}

}

func publish(targetUrl, targetTopic, targetTopicPrefix string, req *http.Request) error {
	topic := targetTopicPrefix + targetTopic

	log.Debugf("Forwarding request from %v to %v", req.URL.String(), topic)
	return nil
}
