package handlers

import (
	"encoding/json"
	"errors"
	"github.com/auth0-community/go-auth0"
	"github.com/mitchellh/mapstructure"
	"github.com/nats-io/go-nats-streaming"
	"github.com/osstotalsoft/bifrost/gateway"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"gopkg.in/square/go-jose.v2"
	"io/ioutil"
	"net/http"
	"time"
)

type NatsConfig struct {
	NatsUrl     string `mapstructure:"nats_url"`
	Cluster     string `mapstructure:"cluster"`
	ClientId    string `mapstructure:"client_id"`
	QGroup      string `mapstructure:"q_group"`
	DurableName string `mapstructure:"durable_name"`
	TopicPrefix string `mapstructure:"topic_prefix"`
}

type NatsEndpointConfig struct {
	Topic string `mapstructure:"topic"`
}

type NatsConnection struct {
	internalConn *stan.Conn
}

type Message struct {
	Headers map[string]interface{}
	Payload map[string]interface{}
}

type CommandResult struct {
	CommandId     uuid.UUID
	CorrelationId uuid.UUID
}

func NewNatsPublisher(config NatsConfig) (gateway.HandlerFunc, NatsConnection) {

	natsConnection, _ := connect(config.NatsUrl, config.ClientId, config.Cluster)

	handlerFunc := func(endpoint gateway.Endpoint) http.Handler {
		var h http.Handler
		var endpointConfig NatsEndpointConfig

		_ = mapstructure.Decode(endpoint.HandlerConfig, &endpointConfig)

		h = http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {

			/*claims, err := getClaims(request)
			if  err != nil {
				http.Error(writer, err.Error(), http.StatusUnauthorized)
				return
			}
			userId := claims["sub"]
			charismaUserId := claims["charisma_userid"]*/

			correlationId := uuid.NewV4()
			commandId := uuid.NewV4()

			headers := map[string]interface{}{
				//"UserId": userId,
				"CharismaUserId": 1, //charismaUserId,
				"CorrelationId":  correlationId,
			}
			payloadChanges := map[string]interface{}{
				"CommandId": commandId,
				"Metadata":  map[string]interface{}{"CreatedDate": time.Now()},
			}

			if err := publish(config.TopicPrefix, endpointConfig.Topic, natsConnection.internalConn, request, headers, payloadChanges); err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			responseBytes, _ := json.Marshal(CommandResult{CommandId: commandId, CorrelationId: correlationId})

			writer.WriteHeader(200)
			writer.Write(responseBytes)
		})

		return h
	}

	return handlerFunc, natsConnection
}

func publish(topicPrefix, targetTopic string, nc *stan.Conn, req *http.Request,
	headers, payloadChanges map[string]interface{}) error {
	topic := topicPrefix + targetTopic

	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		err = errors.New("Nats publisher: could not read request body")
		return err
	}

	envelopeBytes := envelopeMessage(bodyBytes, headers, payloadChanges)

	// Publish the message
	if err := (*nc).Publish(topic, envelopeBytes); err != nil {
		log.Fatal(err)
		return err
	}

	log.Debugf("Forwarding request from %v to %v", req.URL.String(), topic)
	return nil
}

func envelopeMessage(payloadBytes []byte, headers, payloadChanges map[string]interface{}) []byte {

	var payload map[string]interface{}

	_ = json.Unmarshal(payloadBytes, &payload)
	message := Message{
		Headers: headers,
		Payload: payload,
	}

	for k, v := range payloadChanges {
		payload[k] = v
	}

	envelopeBytes, _ := json.Marshal(message)

	return envelopeBytes
}

func connect(natsUrl, clientId, clusterId string) (NatsConnection, error) {
	nc, err := stan.Connect(clusterId, clientId, stan.NatsURL(natsUrl))
	if err != nil {
		log.Fatal(err)
		return NatsConnection{internalConn: nil}, err
	}

	return NatsConnection{internalConn: &nc}, err
}

func (conn *NatsConnection) Close() {
	if conn.internalConn != nil {
		//*conn.internalConn.Flush()
		(*conn.internalConn).Close()
	}
}

func getClaims(req *http.Request) (map[string]interface{}, error) {
	client := auth0.NewJWKClient(auth0.JWKClientOptions{URI: "https://tech0.eu.auth0.com/.well-known/jwks.json"}, nil)
	audience := []string{"http://localhost:8000/api/"}
	configuration := auth0.NewConfiguration(client, audience, "https://tech0.eu.auth0.com/", jose.RS256)
	validator := auth0.NewValidator(configuration, nil)

	token, err := validator.ValidateRequest(req)
	if err != nil {
		log.Errorln("Token is not valid:", token, err)
		return nil, err
	}

	claims := map[string]interface{}{}
	err = validator.Claims(req, token, &claims)
	if err != nil {
		log.Errorln("Invalid claims:", err)
		return nil, err
	}

	return claims, nil
}
