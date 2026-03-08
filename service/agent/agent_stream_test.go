package agent

import (
	"context"
	"errors"
	"testing"

	agentcommon "GopherAI/common/agent"
	"GopherAI/model"
)

func TestEmitEventHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	events := make(chan SSEEvent)
	err := emitEvent(ctx, events, SSEEvent{Type: SSEEventTypeMeta})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestTranslateStreamEventMapsToolCallPayload(t *testing.T) {
	got := translateStreamEvent(agentcommon.StreamEvent{
		Type: agentcommon.StreamEventTypeToolCall,
		ToolCall: &model.ToolCall{
			ToolID:    "tool-1",
			Function:  "knowledge_search",
			Arguments: []byte(`{"query":"gin"}`),
		},
	})

	if got.Type != SSEEventTypeToolCall {
		t.Fatalf("unexpected event type: %q", got.Type)
	}
	if got.ToolID != "tool-1" || got.Function != "knowledge_search" {
		t.Fatalf("unexpected tool call mapping: %#v", got)
	}
	if string(got.Arguments) != `{"query":"gin"}` {
		t.Fatalf("unexpected arguments mapping: %s", string(got.Arguments))
	}
}
