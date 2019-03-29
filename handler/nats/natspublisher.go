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
	NatsUrl              string `mapstructure:"nats_url"`
	Cluster              string `mapstructure:"cluster"`
	ClientId             string `mapstructure:"client_id"`
	QGroup               string `mapstructure:"q_group"`
	DurableName          string `mapstructure:"durable_name"`
	TopicPrefix          string `mapstructure:"topic_prefix"`
	Source               string `mapstructure:"source"`
	transformMessageFunc TransformMessageFunc
	buildResponseFunc    BuildResponseFunc
	logger               log.Logger
}

//EndpointConfig is the NATS specific configuration of the endpoint
type EndpointConfig struct {
	Topic string `mapstructure:"topic"`
}

//CloseConnectionFunc is to be called to close the NATS connection
type CloseConnectionFunc func() error

//TransformMessageFunc transforms a message received in the HTTP request to a format required by the NBB infrastructure.
//It envelopes the message adding the required metadata such as UserId, CorrelationId, MessageId, PublishTime, Source, etc.
type TransformMessageFunc func(messageContext messageContext, requestContext context.Context, payloadBytes []byte) ([]byte, error)

//BuildResponseFunc builds the response that is returned by the Gateway after publishing a message
// The returned data will be written to the HTTP response
type BuildResponseFunc func(messageContext messageContext, requestContext context.Context) ([]byte, error)

type messageContext struct {
	Source     string
	Logger     log.Logger
	Topic      string
	RawPayload []byte
	Headers    map[string]interface{}
}

//NewNatsPublisher creates an instance of the NATS publisher handler.
// It transforms the received HTTP request using the transformMessageFunc into a message, publishes the message to NATS and
// returns the http response built using buildResponseFunc
func NewNatsPublisher(config Config, options ...Option) (handler.Func, CloseConnectionFunc, error) {

	config.transformMessageFunc = NoTransformation
	config.buildResponseFunc = EmptyResponse
	config.logger = log.NewNop()

	config = applyOptions(config, options)

	natsConnection, closeConnectionFunc, err := connect(config.NatsUrl, config.ClientId, config.Cluster, config.logger)
	if err != nil {
		//logger.Error("cannot connect", zap.Error(err))
		return nil, closeConnectionFunc, err
	}

	handlerFunc := func(endpoint abstraction.Endpoint, loggerFactory log.Factory) http.Handler {
		var cfg EndpointConfig
		_ = mapstructure.Decode(endpoint.HandlerConfig, &cfg)

		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			var messageContext = messageContext{Headers: map[string]interface{}{}}
			messageContext.Source = config.Source
			messageContext.Topic = config.TopicPrefix + cfg.Topic
			messageContext.Logger = loggerFactory(request.Context())

			messageBytes, err := ioutil.ReadAll(request.Body)
			if err != nil {
				badRequest(messageContext.Logger, err, "cannot read body", writer)
				return
			}

			messageBytes, err = config.transformMessageFunc(messageContext, request.Context(), messageBytes)
			if err != nil {
				internalServerError(messageContext.Logger, err, "cannot transform", writer)
				return
			}

			if err := natsConnection.Publish(messageContext.Topic, messageBytes); err != nil {
				internalServerError(messageContext.Logger, err, "cannot publish", writer)
				return
			}

			messageContext.Logger.Debug(
				fmt.Sprintf("Forwarding request from %v to %v", request.URL.String(), messageContext.Topic),
				zap.String("request_url", request.URL.String()),
				zap.String("topic", messageContext.Topic))

			responseBytes, err := config.buildResponseFunc(messageContext, request.Context())
			if err != nil {
				internalServerError(messageContext.Logger, err, "build response error", writer)
				return
			}

			if responseBytes != nil {
				_, _ = writer.Write(responseBytes)
			}
		})
	}
	return handlerFunc, closeConnectionFunc, nil
}

func internalServerError(logger log.Logger, err error, msg string, writer http.ResponseWriter) {
	logger.Error(msg, zap.Error(err))
	http.Error(writer, err.Error(), http.StatusInternalServerError)
}

func badRequest(logger log.Logger, err error, msg string, writer http.ResponseWriter) {
	logger.Error(msg, zap.Error(err))
	http.Error(writer, err.Error(), http.StatusBadRequest)
}

//connect opens a streaming NATS connection
func connect(natsUrl, clientId, clusterId string, logger log.Logger) (stan.Conn, CloseConnectionFunc, error) {
	nc, err := stan.Connect(clusterId, clientId+uuid.NewV4().String(), stan.NatsURL(natsUrl))
	if err != nil {
		//logger.Error("cannot connect to nats server", zap.Error(err))
		return nc, nil, err
	}

	return nc, func() error {
		logger.Info("closing nats connection", zap.Error(err))

		err := nc.Close()
		if err != nil {
			logger.Error("cannot close nats connection", zap.Error(err))
		}
		return err
	}, err
}
