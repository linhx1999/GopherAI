package agent

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestBuildHistoryMessageItemsPreservesReasoningContent(t *testing.T) {
	items := buildHistoryMessageItems([]*model.Message{
		{
			SessionID: "sess_1",
			UserName:  "alice",
			Index:     1,
			CreatedAt: time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC),
			Payload:   []byte(`{"role":"assistant","content":"答案","reasoning_content":"先思考"}`),
		},
	})

	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].Index != 1 {
		t.Fatalf("unexpected index: %d", items[0].Index)
	}
	if items[0].Message == nil {
		t.Fatal("expected schema message")
	}
	if items[0].Message.ReasoningContent != "先思考" {
		t.Fatalf("unexpected reasoning content: %q", items[0].Message.ReasoningContent)
	}
	if items[0].Message.Content != "答案" {
		t.Fatalf("unexpected content: %q", items[0].Message.Content)
	}
	if items[0].CreatedAt != "2026-03-10T09:00:00Z" {
		t.Fatalf("unexpected created_at: %q", items[0].CreatedAt)
	}
}
