package nats

import (
	"context"
	"encoding/json"
	"github.com/osstotalsoft/bifrost/abstraction"
	"github.com/satori/go.uuid"
	"testing"
	"time"
)

func TestTransformMessage(t *testing.T) {

	// Arrange
	var payload = map[string]interface{}{
		"myField": "myValue",
	}
	var payloadBytes, _ = json.Marshal(payload)
	var messageContext = messageContext{Source: "src", Headers: map[string]interface{}{}}

	var claimsMap = map[string]interface{}{
		UserIdClaimKey:     "user1",
		CharismaIdClaimKey: 999,
	}
	var requestContext = context.WithValue(context.Background(), abstraction.ContextClaimsKey, claimsMap)
	var response Message

	// Act
	responseBytes, _ := NBBTransformMessage(messageContext, requestContext, payloadBytes)

	// Assert
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		t.Fatal(err.Error())
	}

	if response.Headers == nil {
		t.Fatal("headers not present in the message")
	} else {
		if userId, ok := response.Headers[UserIdKey]; !ok || userId != "user1" {
			t.Fatal(UserIdKey + " header not present in the message")
		}
		if charismaUserId, ok := response.Headers[CharismaUserIdKey].(float64); !ok || charismaUserId != 999 {
			t.Fatal(CharismaUserIdKey + " header not present in the message")
		}
		if _, ok := response.Headers[CorrelationIdKey]; !ok {
			t.Fatal(CorrelationIdKey + " header not present in the message")
		}
		if _, ok := response.Headers[MessageIdKey]; !ok {
			t.Fatal(MessageIdKey + " header not present in the message")
		}
		if _, ok := response.Headers[PublishTimeKey]; !ok {
			t.Fatal(PublishTimeKey + " header not present in the message")
		}
		if source, ok := response.Headers[SourceKey]; !ok || source != "src" {
			t.Fatal(SourceKey + " header not present in the message")
		}
	}

	if response.Payload == nil {
		t.Fatal("payload not present in the message")
	} else {
		if _, ok := response.Payload[CommandIdKey]; !ok {
			t.Fatal(CommandIdKey + " not present in the payload")
		}
		if metadata, ok := response.Payload[MetadataKey].(map[string]interface{}); !ok {
			t.Error(MetadataKey + " header not present in the payload")
			if _, ok := metadata[CreationDateKey].(time.Time); !ok {
				t.Fatal(CreationDateKey + " metadata not present in the message")
			}
		}
	}

	if _, ok := messageContext.Headers[CommandIdKey]; !ok {
		t.Fatal(CommandIdKey + " not present in the message context")
	}

	if _, ok := messageContext.Headers[CorrelationIdKey]; !ok {
		t.Fatal(CorrelationIdKey + " not present in the message context")
	}
}

func TestBuildResponse(t *testing.T) {

	// Arrange
	var correlationId = uuid.NewV4()
	var commandId = uuid.NewV4()

	var messageContext = messageContext{Headers: map[string]interface{}{
		CorrelationIdKey: correlationId,
		CommandIdKey:     commandId,
	}}
	var requestContext = context.WithValue(context.Background(), abstraction.ContextClaimsKey, nil)
	var expectedResponse, _ = json.Marshal(CommandResult{CommandId: commandId, CorrelationId: correlationId})

	// Act
	resp, _ := NBBBuildResponse(messageContext, requestContext)

	// Assert
	if string(resp) != string(expectedResponse) {
		t.Fatal("Response does not match expected value")
	}
}
