package deepagent

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestStreamHandlerReturnsErrorWhenMessageMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/deep-agent/stream", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	StreamHandler(c)

	body := recorder.Body.String()
	if !strings.Contains(body, `"type":"response.error"`) {
		t.Fatalf("expected error payload, got %q", body)
	}
	if !strings.Contains(body, `"code":2001`) {
		t.Fatalf("expected invalid params code, got %q", body)
	}
}

func TestStreamHandlerReturnsErrorWhenSessionIDMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := newCloseNotifyRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/deep-agent/stream", bytes.NewBufferString(`{"message":"你好"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	StreamHandler(c)

	body := recorder.Body.String()
	if !strings.Contains(body, `"type":"response.error"`) {
		t.Fatalf("expected error payload, got %q", body)
	}
	if !strings.Contains(body, `"message":"session_id is required"`) {
		t.Fatalf("expected missing session message, got %q", body)
	}
}
