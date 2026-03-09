package agent

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestCollectStreamMessageConcatsChunksAndEmitsSyntheticFinish(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{
			Role:             schema.Assistant,
			ReasoningContent: "先分析",
		},
		{
			Role:    schema.Assistant,
			Content: "答案",
		},
	})

	var emitted []*schema.Message
	full, err := collectStreamMessage(reader, func(msg *schema.Message) error {
		emitted = append(emitted, msg)
		return nil
	})
	if err != nil {
		t.Fatalf("collectStreamMessage returned error: %v", err)
	}
	if full == nil {
		t.Fatal("expected full message")
	}
	if full.Content != "答案" {
		t.Fatalf("unexpected content: %q", full.Content)
	}
	if full.ReasoningContent != "先分析" {
		t.Fatalf("unexpected reasoning content: %q", full.ReasoningContent)
	}
	if full.ResponseMeta == nil || full.ResponseMeta.FinishReason != "stop" {
		t.Fatalf("expected synthetic finish reason, got %#v", full.ResponseMeta)
	}
	if len(emitted) != 3 {
		t.Fatalf("expected 3 emitted chunks, got %d", len(emitted))
	}
	if emitted[2].ResponseMeta == nil || emitted[2].ResponseMeta.FinishReason != "stop" {
		t.Fatalf("expected last emitted chunk to be synthetic finish, got %#v", emitted[2].ResponseMeta)
	}
}

func TestCollectStreamMessageKeepsExistingFinishReason(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{
			Role:    schema.Tool,
			Content: `{"result":"ok"}`,
			ResponseMeta: &schema.ResponseMeta{
				FinishReason: "stop",
			},
		},
	})

	var finishReasons []string
	full, err := collectStreamMessage(reader, func(msg *schema.Message) error {
		if msg.ResponseMeta != nil {
			finishReasons = append(finishReasons, msg.ResponseMeta.FinishReason)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("collectStreamMessage returned error: %v", err)
	}
	if full == nil || full.ResponseMeta == nil || full.ResponseMeta.FinishReason != "stop" {
		t.Fatalf("unexpected full message response meta: %#v", full)
	}
	if !reflect.DeepEqual(finishReasons, []string{"stop"}) {
		t.Fatalf("unexpected finish reasons: %v", finishReasons)
	}
}

func TestCollectStreamMessageStopsOnSinkError(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{Role: schema.Assistant, Content: "答"},
	})

	expectedErr := errors.New("sink failed")
	_, err := collectStreamMessage(reader, func(msg *schema.Message) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected sink error, got %v", err)
	}
}
