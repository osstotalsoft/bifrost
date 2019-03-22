package nats

import (
	"context"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/nats-io/go-nats-streaming"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/osstotalsoft/bifrost/handler"
	"github.com/osstotalsoft/bifrost/log"
	"github.com/satori/go.uuid"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

//Config is the global NATS configuration
type Config struct {
	NatsUrl     string `mapstructure:"nats_url"`
	Cluster     string `mapstructure:"cluster"`
	ClientId    string `mapstructure:"client_id"`
	QGroup      string `mapstructure:"q_group"`
	DurableName string `mapstructure:"durable_name"`
	TopicPrefix string `mapstructure:"topic_prefix"`
	Source      string `mapstructure:"source"`
}

//EndpointConfig is the NATS specific configuration of the endpoint
type EndpointConfig struct {
	Topic string `mapstructure:"topic"`
}

//CloseConnectionFunc is to be called to close the NATS connection
type CloseConnectionFunc func()

//TransformMessageFunc transforms a message received in the HTTP request to a format required by the NBB infrastructure.
//It envelopes the message adding the required metadata such as UserId, CorrelationId, MessageId, PublishTime, Source, etc.
type TransformMessageFunc func(payloadBytes []byte, messageContext map[string]interface{}, requestContext context.Context) ([]byte, error)

//BuildResponseFunc builds the response that is returned by the Gateway after publishing a message
// The returned data will be written to the HTTP response
type BuildResponseFunc func(messageContext map[string]interface{}, requestContext context.Context) ([]byte, error)

//NewNatsPublisher creates an instance of the NATS publisher handler.
// It transforms the received HTTP request using the transformMessageFunc into a message, publishes the message to NATS and
// returns the http response built using buildResponseFunc
func NewNatsPublisher(config Config, transformMessageFunc TransformMessageFunc, buildResponseFunc BuildResponseFunc, logger log.Logger) (handler.Func, CloseConnectionFunc) {

	natsConnection, closeConnectionFunc, err := connect(config.NatsUrl, config.ClientId, config.Cluster, logger)
	if err != nil {
		logger.Error("cannot connect", zap.Error(err))
		return nil, closeConnectionFunc
	}

	handlerFunc := func(endpoint abstraction.Endpoint, loggerFactory log.Factory) http.Handler {
		var cfg EndpointConfig

		_ = mapstructure.Decode(endpoint.HandlerConfig, &cfg)

		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			var messageContext = map[string]interface{}{}
			messageContext[SourceKey] = config.Source
			topic := config.TopicPrefix + cfg.Topic
			logger := loggerFactory(request.Context())

			messageBytes, err := ioutil.ReadAll(request.Body)
			if err != nil {
				logger.Error("cannot read body", zap.Error(err))
				http.Error(writer, "Bad request", http.StatusBadRequest)
				return
			}

			if transformMessageFunc != nil {
				messageBytes, err = transformMessageFunc(messageBytes, messageContext, request.Context())
				if err != nil {
					logger.Error("cannot transform", zap.Error(err))
					http.Error(writer, "Internal server error", http.StatusInternalServerError)
					return
				}
			}

			if err := natsConnection.Publish(topic, messageBytes); err != nil {
				logger.Error("cannot publish", zap.Error(err))
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			logger.Debug(fmt.Sprintf("Forwarding request from %v to %v", request.URL.String(), topic),
				zap.String("request_url", request.URL.String()), zap.String("topic", topic))

			if buildResponseFunc != nil {
				responseBytes, err := buildResponseFunc(messageContext, request.Context())

				if err != nil {
					logger.Error("build response error", zap.Error(err))
					http.Error(writer, "Internal server error", http.StatusInternalServerError)
					return
				}

				_, _ = writer.Write(responseBytes)
			}
		})
	}
	return handlerFunc, closeConnectionFunc
}

//connect opens a streaming NATS connection
func connect(natsUrl, clientId, clusterId string, logger log.Logger) (stan.Conn, CloseConnectionFunc, error) {
	nc, err := stan.Connect(clusterId, clientId+uuid.NewV4().String(), stan.NatsURL(natsUrl))
	if err != nil {
		logger.Error("cannot connect to nats server", zap.Error(err))
		return nc, func() {}, err
	}

	return nc, func() {
		err := nc.Close()
		if err != nil {
			logger.Error("cannot close nats connection", zap.Error(err))
		}
	}, err
}
