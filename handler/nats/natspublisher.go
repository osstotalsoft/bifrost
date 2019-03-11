package nats

import (
	"context"
	"github.com/mitchellh/mapstructure"
	"github.com/nats-io/go-nats-streaming"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

type Config struct {
	NatsUrl     string `mapstructure:"nats_url"`
	Cluster     string `mapstructure:"cluster"`
	ClientId    string `mapstructure:"client_id"`
	QGroup      string `mapstructure:"q_group"`
	DurableName string `mapstructure:"durable_name"`
	TopicPrefix string `mapstructure:"topic_prefix"`
}

type EndpointConfig struct {
	Topic string `mapstructure:"topic"`
}

type CloseConnectionFunc func()

type TransformMessageFunc func(payloadBytes []byte, messageContext map[string]interface{}, requestContext context.Context) ([]byte, error)
type BuildResponseFunc func(messageContext map[string]interface{}, requestContext context.Context) ([]byte, error)

func NewNatsPublisher(config Config, transformMessageFunc TransformMessageFunc, buildResponseFunc BuildResponseFunc) (handler.Func, CloseConnectionFunc) {

	natsConnection, closeConnectionFunc, err := connect(config.NatsUrl, config.ClientId, config.Cluster)
	if err != nil {
		log.Error(err)
		return nil, closeConnectionFunc
	}

	handlerFunc := func(endpoint abstraction.Endpoint) http.Handler {
		var h http.Handler
		var cfg EndpointConfig

		_ = mapstructure.Decode(endpoint.HandlerConfig, &cfg)

		h = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			var messageContext = map[string]interface{}{}
			topic := config.TopicPrefix + cfg.Topic

			messageBytes, err := ioutil.ReadAll(request.Body)
			if err != nil {
				log.Error(err)
				http.Error(writer, "Bad request", http.StatusBadRequest)
				return
			}

			if transformMessageFunc != nil {
				messageBytes, err = transformMessageFunc(messageBytes, messageContext, request.Context())
				if err != nil {
					log.Error(err)
					http.Error(writer, "Internal server error", http.StatusInternalServerError)
					return
				}
			}

			if err := natsConnection.Publish(topic, messageBytes); err != nil {
				log.Error(err)
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Tracef("Forwarding request from %v to %v", request.URL.String(), topic)

			if buildResponseFunc != nil {
				responseBytes, err := buildResponseFunc(messageContext, request.Context())

				if err != nil {
					log.Error(err)
					http.Error(writer, "Internal server error", http.StatusInternalServerError)
					return
				}

				_, _ = writer.Write(responseBytes)
			}
		})

		return h
	}
	return handlerFunc, closeConnectionFunc
}

func connect(natsUrl, clientId, clusterId string) (stan.Conn, CloseConnectionFunc, error) {
	nc, err := stan.Connect(clusterId, clientId+uuid.NewV4().String(), stan.NatsURL(natsUrl))
	if err != nil {
		log.Fatal(err)
		return nc, func() {}, err
	}

	return nc, func() {
		err := nc.Close()
		log.Error(err)
	}, err
}
