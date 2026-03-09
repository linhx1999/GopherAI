package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/schema"

	"GopherAI/model"
)

func TestEmitEventHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	events := make(chan StreamEvent)
	err := emitEvent(ctx, events, StreamEvent{
		Meta: &StreamMeta{Type: StreamPayloadTypeMeta},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestNewErrorEventBuildsErrorPayload(t *testing.T) {
	event := NewErrorEvent("bad request")
	if event.Error == nil {
		t.Fatal("expected error payload")
	}
	if event.Error.Type != StreamPayloadTypeError {
		t.Fatalf("unexpected error type: %q", event.Error.Type)
	}
	if event.Error.Message != "bad request" {
		t.Fatalf("unexpected error message: %q", event.Error.Message)
	}
}

func TestBuildConversationMessagesUsesStoredSchemaPayload(t *testing.T) {
	history := []*model.Message{
		{
			Payload: []byte(`{"role":"user","content":"你好"}`),
		},
	}

	messages := buildConversationMessages(history, &schema.Message{
		Role:    schema.User,
		Content: "继续",
	}, false)

	if len(messages) != 3 {
		t.Fatalf("unexpected message count: %d", len(messages))
	}
	if messages[1].Role != schema.User || messages[1].Content != "你好" {
		t.Fatalf("unexpected history payload: %#v", messages[1])
	}
	if messages[2].Content != "继续" {
		t.Fatalf("unexpected user message: %#v", messages[2])
	}
}
