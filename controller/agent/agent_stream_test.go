package agent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"

	"GopherAI/common/code"
	agentService "GopherAI/service/agent"
)

type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
	closeCh chan bool
}

func newCloseNotifyRecorder() *closeNotifyRecorder {
	return &closeNotifyRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closeCh:          make(chan bool, 1),
	}
}

func (r *closeNotifyRecorder) CloseNotify() <-chan bool {
	return r.closeCh
}

func TestStreamLoopUsesGinSSEventAndDoneMarker(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/stream", nil)

	streamSSE(c, oneShotSSEStream(agentService.StreamEvent{
		Meta: &agentService.StreamMeta{
			Type:         agentService.StreamPayloadTypeMeta,
			SessionID:    "sess_1",
			MessageIndex: 1,
		},
	}))

	body := recorder.Body.String()
	if !strings.Contains(body, "event:message") {
		t.Fatalf("expected gin SSE event line, got %q", body)
	}
	if !strings.Contains(body, `"type":"meta"`) {
		t.Fatalf("expected meta payload, got %q", body)
	}
	if !strings.Contains(body, `"session_id":"sess_1"`) {
		t.Fatalf("expected session_id in payload, got %q", body)
	}
	if !strings.Contains(body, "data:[DONE]") {
		t.Fatalf("expected done marker, got %q", body)
	}
}

func TestStreamLoopWritesSchemaMessagePayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/stream", nil)

	streamSSE(c, oneShotSSEStream(agentService.StreamEvent{
		Message: &schema.Message{
			Role:             schema.Assistant,
			Content:          "答案",
			ReasoningContent: "先想",
		},
	}))

	body := recorder.Body.String()
	if !strings.Contains(body, `"role":"assistant"`) {
		t.Fatalf("expected assistant payload, got %q", body)
	}
	if !strings.Contains(body, `"reasoning_content":"先想"`) {
		t.Fatalf("expected reasoning content, got %q", body)
	}
}

func TestHandleStreamRequestReturnsErrorEventWhenMessageMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/stream", nil)

	events := resolveStreamSource(c.Request.Context(), &AgentRequest{Stream: true}, "alice")
	event, ok := <-events
	if !ok {
		t.Fatal("expected error event")
	}
	if event.Error == nil {
		t.Fatal("expected error payload")
	}
	if event.Error.Type != agentService.StreamPayloadTypeError {
		t.Fatalf("expected error type, got %q", event.Error.Type)
	}
	if event.Error.Message != "message is required" {
		t.Fatalf("expected message validation error, got %q", event.Error.Message)
	}
	if _, ok := <-events; ok {
		t.Fatal("expected channel to close after single error event")
	}
}

func TestErrorEventStreamCanBeConsumedByGinStream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/stream", nil)

	streamSSE(c, errorEventStream(code.CodeInvalidParams, "bad request"))

	body := recorder.Body.String()
	if !strings.Contains(body, "event:message") {
		t.Fatalf("expected gin SSE event line, got %q", body)
	}
	if !strings.Contains(body, `"type":"error"`) {
		t.Fatalf("expected error payload, got %q", body)
	}
	if !strings.Contains(body, `"message":"bad request"`) {
		t.Fatalf("expected error message in payload, got %q", body)
	}
	if !strings.Contains(body, "data:[DONE]") {
		t.Fatalf("expected done marker, got %q", body)
	}
}
