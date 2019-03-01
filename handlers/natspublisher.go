package handlers

import (
	"github.com/osstotalsoft/bifrost/gateway"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type NatsConfig struct {
	natsUrl     string `json:"handlers.nats.nats_url"`
	cluster     string `json:"handlers.nats.cluster"`
	clientId    string `json:"handlers.nats.client_id"`
	qGroup      string `json:"handlers.nats.q_group"`
	durableName string `json:"handlers.nats.durable_name"`
}

func NewNatsPublisher(config NatsConfig) gateway.HandlerFunc {

	return func(endpoint gateway.Endpoint) http.Handler {
		var h http.Handler
		h = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			if err := publish("", "", "", request); err != nil {
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
