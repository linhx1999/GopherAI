package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"

	"GopherAI/common/code"
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
	})

	if len(messages) != 2 {
		t.Fatalf("unexpected message count: %d", len(messages))
	}
	if messages[0].Role != schema.User || messages[0].Content != "你好" {
		t.Fatalf("unexpected history payload: %#v", messages[0])
	}
	if messages[1].Content != "继续" {
		t.Fatalf("unexpected user message: %#v", messages[1])
	}
}

func TestBuildAgentInstructionReflectsToolState(t *testing.T) {
	withTools := buildAgentInstruction(true)
	withoutTools := buildAgentInstruction(false)

	if !strings.Contains(withTools, "当前对话已启用工具") {
		t.Fatalf("expected tool-enabled instruction, got %q", withTools)
	}
	if !strings.Contains(withoutTools, "当前对话未启用工具") {
		t.Fatalf("expected tool-disabled instruction, got %q", withoutTools)
	}
}

func TestBuildHistoryMessageItemsPreservesReasoningContent(t *testing.T) {
	items := buildHistoryMessageItems([]*model.Message{
		{
			Model:     gorm.Model{CreatedAt: time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)},
			MessageID: "msg_1",
			Index:     1,
			Payload:   []byte(`{"role":"assistant","content":"答案","reasoning_content":"先思考"}`),
		},
	})

	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].MessageID != "msg_1" {
		t.Fatalf("unexpected message_id: %q", items[0].MessageID)
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

func TestIsRequestCanceledDetectsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if !isRequestCanceled(ctx, nil) {
		t.Fatal("expected canceled context to be detected")
	}
}

func TestIsRequestCanceledDetectsWrappedCancellationError(t *testing.T) {
	err := fmt.Errorf("llm generate failed: %w", context.Canceled)
	if !isRequestCanceled(context.Background(), err) {
		t.Fatal("expected wrapped cancellation error to be detected")
	}
}

func TestIsRequestCanceledDetectsCancellationText(t *testing.T) {
	err := errors.New("[NodeRunError] failed to create chat completion: Post \"https://example.com\": context canceled")
	if !isRequestCanceled(context.Background(), err) {
		t.Fatal("expected cancellation text to be detected")
	}
}

func TestResolveExecutionSessionRejectsMissingSessionWhenCreationDisabled(t *testing.T) {
	session, code_ := resolveExecutionSession(1, "", "你好", false)
	if session != nil {
		t.Fatalf("expected nil session, got %#v", session)
	}
	if code_ != code.CodeInvalidParams {
		t.Fatalf("expected invalid params code, got %d", code_)
	}
}

func TestSyncSessionTitleWithFirstMessageSkipsNonPlaceholderSession(t *testing.T) {
	session := &model.Session{Title: "已有标题"}
	code_ := syncSessionTitleWithFirstMessage(session, 1, nil, "新的首条消息")
	if code_ != code.CodeSuccess {
		t.Fatalf("expected success code, got %d", code_)
	}
	if session.Title != "已有标题" {
		t.Fatalf("expected title to stay unchanged, got %q", session.Title)
	}
}
