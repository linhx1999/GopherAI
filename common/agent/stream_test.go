package agent

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestConsumeStreamThinkingModeOrdering(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{
			ReasoningContent: "先",
			ToolCalls: []schema.ToolCall{{
				ID:   "tool-1",
				Type: "function",
				Function: schema.FunctionCall{
					Name:      "knowledge_search",
					Arguments: `{"query":"golang"}`,
				},
			}},
		},
		{ReasoningContent: "想"},
		{Content: "答"},
		{Content: "案"},
	})

	var events []string
	result, err := ConsumeStream(reader, true, func(event StreamEvent) error {
		switch event.Type {
		case StreamEventTypeToolCall:
			events = append(events, "tool_call:"+event.ToolCall.Function)
		case StreamEventTypeReasoningDelta:
			events = append(events, "reasoning_delta:"+event.Content)
		case StreamEventTypeReasoningEnd:
			events = append(events, "reasoning_end")
		case StreamEventTypeContentDelta:
			events = append(events, "content_delta:"+event.Content)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("ConsumeStream returned error: %v", err)
	}

	wantEvents := []string{
		"tool_call:knowledge_search",
		"reasoning_delta:先",
		"reasoning_delta:想",
		"reasoning_end",
		"content_delta:答",
		"content_delta:案",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events mismatch, got %v want %v", events, wantEvents)
	}
	if result.Content != "答案" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if result.ReasoningContent != "先想" {
		t.Fatalf("unexpected reasoning content: %q", result.ReasoningContent)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("unexpected tool call count: %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].Function != "knowledge_search" {
		t.Fatalf("unexpected tool call function: %q", result.ToolCalls[0].Function)
	}
	if string(result.ToolCalls[0].Arguments) != `{"query":"golang"}` {
		t.Fatalf("unexpected tool call arguments: %s", string(result.ToolCalls[0].Arguments))
	}
}

func TestConsumeStreamReasoningOnlyClosesOnEOF(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{ReasoningContent: "先确认约束"},
	})

	var events []StreamEventType
	result, err := ConsumeStream(reader, true, func(event StreamEvent) error {
		events = append(events, event.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("ConsumeStream returned error: %v", err)
	}

	want := []StreamEventType{
		StreamEventTypeReasoningDelta,
		StreamEventTypeReasoningEnd,
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events mismatch, got %v want %v", events, want)
	}
	if result.Content != "" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if result.ReasoningContent != "先确认约束" {
		t.Fatalf("unexpected reasoning content: %q", result.ReasoningContent)
	}
}

func TestConsumeStreamNonThinkingModeDoesNotEmitReasoningEnd(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{ReasoningContent: "先想"},
		{Content: "回答"},
	})

	var events []StreamEventType
	_, err := ConsumeStream(reader, false, func(event StreamEvent) error {
		events = append(events, event.Type)
		return nil
	})
	if err != nil {
		t.Fatalf("ConsumeStream returned error: %v", err)
	}

	want := []StreamEventType{
		StreamEventTypeReasoningDelta,
		StreamEventTypeContentDelta,
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events mismatch, got %v want %v", events, want)
	}
}

func TestConsumeStreamStopsOnSinkError(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{Content: "答"},
		{Content: "案"},
	})

	expectedErr := errors.New("sink failed")
	_, err := ConsumeStream(reader, false, func(event StreamEvent) error {
		if event.Type == StreamEventTypeContentDelta {
			return expectedErr
		}
		return nil
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected sink error, got %v", err)
	}
}
