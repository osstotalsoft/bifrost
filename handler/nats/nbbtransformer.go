package nats

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/satori/go.uuid"
	"time"
)

const (
	CorrelationIdKey   = "nbb-correlationId"
	MessageIdKey       = "nbb-messageId"
	PublishTimeKey     = "nbb-publishTime"
	SourceKey          = "nbb-source"
	CommandIdKey       = "CommandId"
	UserIdKey          = "UserId"
	CharismaUserIdKey  = "CharismaUserId"
	MetadataKey        = "Metadata"
	CreationDateKey    = "CreationDate"
	UserIdClaimKey     = "sub"
	CharismaIdClaimKey = "charisma_user_id"
)

//Message is the structure of the message envelope to be published
type Message struct {
	Headers map[string]interface{}
	Payload map[string]interface{}
}

//CommandResult is the structure to be returned in the HTTP response
type CommandResult struct {
	CommandId     uuid.UUID
	CorrelationId uuid.UUID
}

//TransformMessage transforms a message received in the HTTP request to a format required by the NBB infrastructure.
// It envelopes the message adding the required metadata such as UserId, CorrelationId, MessageId, PublishTime, Source, etc.
func NBBTransformMessage(messageContext messageContext, requestContext context.Context, payloadBytes []byte) ([]byte, error) {
	claims, err := getClaims(requestContext)
	if err != nil {
		return nil, err
	}

	userId, ok := claims[UserIdClaimKey]
	if !ok {
		return nil, errors.New(UserIdClaimKey + " claim not found")
	}

	charismaUserId, ok := claims[CharismaIdClaimKey]
	if !ok {
		return nil, errors.New(CharismaIdClaimKey + " claim not found")
	}

	correlationId := uuid.NewV4()
	commandId := uuid.NewV4()
	now := time.Now()

	headers := map[string]interface{}{
		UserIdKey:         userId,
		CharismaUserIdKey: charismaUserId,
		CorrelationIdKey:  correlationId,
		MessageIdKey:      uuid.NewV4(),
		SourceKey:         messageContext.Source,
		PublishTimeKey:    now,
	}
	payloadChanges := map[string]interface{}{
		CommandIdKey: commandId,
		MetadataKey:  map[string]interface{}{CreationDateKey: now},
	}

	messageContext.Headers[CorrelationIdKey] = correlationId
	messageContext.Headers[CommandIdKey] = commandId

	return envelopeMessage(payloadBytes, headers, payloadChanges), nil
}

//BuildResponse builds the response that is returned by the Gateway after publishing a message
func NBBBuildResponse(messageContext messageContext, requestContext context.Context) ([]byte, error) {

	correlationId, ok := messageContext.Headers[CorrelationIdKey].(uuid.UUID)
	if !ok {
		return nil, errors.New("correlation id not found in message context")
	}

	commandId, ok := messageContext.Headers[CommandIdKey].(uuid.UUID)
	if !ok {
		return nil, errors.New("command id not found in message context")
	}

	responseBytes, err := json.Marshal(CommandResult{CommandId: commandId, CorrelationId: correlationId})

	return responseBytes, err
}

//getClaims get the claims map stored in the context
func getClaims(context context.Context) (map[string]interface{}, error) {
	claims, ok := context.Value(abstraction.ContextClaimsKey).(map[string]interface{})
	if !ok {
		return nil, errors.New("claims not present or not authenticated")
	}

	return claims, nil
}

//envelopeMessage envelopes a message payload with the headers specified and applies changes/additions to the payload
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
