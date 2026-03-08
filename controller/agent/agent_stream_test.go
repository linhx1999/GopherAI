package agent

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func consumeStreamWithDone(c *gin.Context, events <-chan agentService.SSEEvent) {
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case msg, ok := <-events:
			if !ok {
				c.SSEvent(ginSSEEventName, "[DONE]")
				return false
			}
			c.SSEvent(ginSSEEventName, msg)
			return true
		}
	})
}

func TestStreamLoopUsesGinSSEventAndDoneMarker(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/stream", nil)

	consumeStreamWithDone(c, singleSSEvent(agentService.SSEEvent{
		Type:         agentService.SSEEventTypeMeta,
		SessionID:    "sess_1",
		MessageIndex: 1,
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

func TestHandleStreamRequestReturnsErrorEventWhenMessageMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/stream", nil)

	events := make(chan agentService.SSEEvent)
	go handleStreamRequest(c, events, &AgentRequest{Stream: true}, "alice")

	event, ok := <-events
	if !ok {
		t.Fatal("expected error event")
	}
	if event.Type != agentService.SSEEventTypeError {
		t.Fatalf("expected error type, got %q", event.Type)
	}
	if event.Message != "message is required" {
		t.Fatalf("expected message validation error, got %q", event.Message)
	}
	if _, ok := <-events; ok {
		t.Fatal("expected channel to close after single error event")
	}
}

func TestHandleStreamRequestReturnsErrorEventWhenRegenerateSessionMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/stream", nil)

	from := 3
	events := make(chan agentService.SSEEvent)
	go handleStreamRequest(c, events, &AgentRequest{
		Stream:         true,
		RegenerateFrom: &from,
	}, "alice")

	event, ok := <-events
	if !ok {
		t.Fatal("expected error event")
	}
	if event.Type != agentService.SSEEventTypeError {
		t.Fatalf("expected error type, got %q", event.Type)
	}
	if event.Message != "session_id is required for regenerate" {
		t.Fatalf("expected regenerate validation error, got %q", event.Message)
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

	consumeStreamWithDone(c, errorEventStream(code.CodeInvalidParams, "bad request"))

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
