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

// The structure of the message envelope to be published
type Message struct {
	Headers map[string]interface{}
	Payload map[string]interface{}
}

// The structure to be returned in the HTTP response
type CommandResult struct {
	CommandId     uuid.UUID
	CorrelationId uuid.UUID
}

// The function transforms a message received in the HTTP request to a format required by the NBB infrastructure.
// It envelopes the message adding the required metadata such as UserId, CorrelationId, MessageId, PublishTime, Source, etc.
func TransformMessage(payloadBytes []byte, messageContext map[string]interface{}, requestContext context.Context) ([]byte, error) {
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
		SourceKey:         messageContext[SourceKey],
		PublishTimeKey:    now,
	}
	payloadChanges := map[string]interface{}{
		CommandIdKey: commandId,
		MetadataKey:  map[string]interface{}{CreationDateKey: now},
	}

	messageContext[CorrelationIdKey] = correlationId
	messageContext[CommandIdKey] = commandId

	return envelopeMessage(payloadBytes, headers, payloadChanges), nil
}

// The function builds the response that is returned by the Gateway after publishing a message
func BuildResponse(messageContext map[string]interface{}, requestContext context.Context) ([]byte, error) {

	correlationId, ok := messageContext[CorrelationIdKey].(uuid.UUID)
	if !ok {
		return nil, errors.New("correlation id not found in message context")
	}

	commandId, ok := messageContext[CommandIdKey].(uuid.UUID)
	if !ok {
		return nil, errors.New("command id not found in message context")
	}

	responseBytes, err := json.Marshal(CommandResult{CommandId: commandId, CorrelationId: correlationId})

	return responseBytes, err
}

// Get the claims map stored in the context
func getClaims(context context.Context) (map[string]interface{}, error) {
	claims, ok := context.Value(abstraction.ContextClaimsKey).(map[string]interface{})
	if !ok {
		return nil, errors.New("claims not present or not authenticated")
	}

	return claims, nil
}

// Envelopes a message payload with the headers specified and applies changes/additions to the payload
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
