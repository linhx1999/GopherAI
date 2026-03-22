package agent

import (
	"bytes"
	"fmt"
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

	streamSSE(c, oneShotSSEStream(agentService.NewSuccessEvent(agentService.StreamEventTypeResponseCreated, nil)))

	body := recorder.Body.String()
	if !strings.Contains(body, "event:message") {
		t.Fatalf("expected gin SSE event line, got %q", body)
	}
	if !strings.Contains(body, `"type":"response.created"`) {
		t.Fatalf("expected created payload, got %q", body)
	}
	if !strings.Contains(body, `"code":1000`) {
		t.Fatalf("expected success code, got %q", body)
	}
}

func TestStreamLoopWritesSchemaMessagePayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/stream", nil)

	streamSSE(c, oneShotSSEStream(agentService.NewSuccessEvent(agentService.StreamEventTypeResponseMessageDelta, &agentService.StreamDeltaResponse{
		Delta: &schema.Message{
			Role:             schema.Assistant,
			Content:          "答案",
			ReasoningContent: "先想",
		},
	})))

	body := recorder.Body.String()
	if !strings.Contains(body, `"type":"response.message.delta"`) {
		t.Fatalf("expected delta type, got %q", body)
	}
	if !strings.Contains(body, `"role":"assistant"`) {
		t.Fatalf("expected assistant payload, got %q", body)
	}
	if !strings.Contains(body, `"reasoning_content":"先想"`) {
		t.Fatalf("expected reasoning content, got %q", body)
	}
}

func TestStreamHandlerReturnsErrorEventWhenMessageMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/stream", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	StreamHandler(c)

	body := recorder.Body.String()
	if !strings.Contains(body, `"type":"response.error"`) {
		t.Fatalf("expected error payload, got %q", body)
	}
	if !strings.Contains(body, `"message":"Key: 'ChatRequest.Message' Error:Field validation for 'Message' failed on the 'required' tag"`) {
		t.Fatalf("expected validation error message, got %q", body)
	}
	if !strings.Contains(body, `"code":2001`) {
		t.Fatalf("expected invalid params code, got %q", body)
	}
}

func TestStreamHandlerReturnsErrorEventWhenSessionIDMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/stream", bytes.NewBufferString(`{"message":"你好"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	StreamHandler(c)

	body := recorder.Body.String()
	if !strings.Contains(body, `"type":"response.error"`) {
		t.Fatalf("expected error payload, got %q", body)
	}
	if !strings.Contains(body, `"message":"session_id is required"`) {
		t.Fatalf("expected session_id required message, got %q", body)
	}
	if !strings.Contains(body, `"code":2001`) {
		t.Fatalf("expected invalid params code, got %q", body)
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
	if !strings.Contains(body, `"type":"response.error"`) {
		t.Fatalf("expected error payload, got %q", body)
	}
	if !strings.Contains(body, `"message":"bad request"`) {
		t.Fatalf("expected error message in payload, got %q", body)
	}
	if !strings.Contains(body, `"code":2001`) {
		t.Fatalf("expected invalid params code, got %q", body)
	}
}

func TestGenerateHandlerReturnsJSONErrorWhenMessageMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/generate", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	GenerateHandler(c)

	body := recorder.Body.String()
	if !strings.Contains(body, fmt.Sprintf(`"code":%d`, code.CodeInvalidParams)) {
		t.Fatalf("expected invalid params code, got %q", body)
	}
	if strings.Contains(body, `event:message`) {
		t.Fatalf("expected JSON response, got SSE payload %q", body)
	}
}
