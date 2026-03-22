package deepagent

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"GopherAI/common/code"
	"GopherAI/controller"
	agentService "GopherAI/service/agent"
	deepagentService "GopherAI/service/deepagent"
)

const ginSSEEventName = "message"

type ChatRequest struct {
	SessionID    string   `json:"session_id"`
	Message      string   `json:"message" binding:"required"`
	Tools        []string `json:"tools"`
	MCPServerIDs []string `json:"mcp_server_ids"`
	ThinkingMode bool     `json:"thinking_mode"`
}

func GenerateHandler(c *gin.Context) {
	req, err := parseChatRequest(c)
	if err != nil {
		writeCodeResponse(c, code.CodeInvalidParams)
		return
	}

	result, code_ := agentService.DeepGenerate(
		c.Request.Context(),
		c.GetUint("userID"),
		c.GetString("userUUID"),
		req.SessionID,
		req.Message,
		req.Tools,
		req.MCPServerIDs,
		req.ThinkingMode,
	)
	if code_ != code.CodeSuccess {
		if c.Request.Context().Err() != nil {
			return
		}
		writeCodeResponse(c, code_)
		return
	}
	if c.Request.Context().Err() != nil {
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: result,
	})
}

func StreamHandler(c *gin.Context) {
	req, err := parseChatRequest(c)
	setSSEHeaders(c)
	if err != nil {
		streamSSE(c, errorEventStream(code.CodeInvalidParams, err.Error()))
		return
	}
	if req.SessionID == "" {
		streamSSE(c, errorEventStream(code.CodeInvalidParams, "session_id is required"))
		return
	}

	stream, code_ := agentService.DeepStream(
		c.Request.Context(),
		c.GetUint("userID"),
		c.GetString("userUUID"),
		req.SessionID,
		req.Message,
		req.Tools,
		req.MCPServerIDs,
		req.ThinkingMode,
	)
	if code_ != code.CodeSuccess {
		streamSSE(c, errorEventStream(code_, code_.Msg()))
		return
	}

	streamSSE(c, stream.Events)
}

func GetRuntime(c *gin.Context) {
	runtime, code_ := deepagentService.GetRuntimeStatus(c.Request.Context(), c.GetUint("userID"), c.GetString("userUUID"))
	if code_ != code.CodeSuccess {
		writeCodeResponse(c, code_)
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: gin.H{"runtime": runtime},
	})
}

func RestartRuntime(c *gin.Context) {
	runtime, code_ := deepagentService.RestartRuntime(c.Request.Context(), c.GetUint("userID"), c.GetString("userUUID"))
	if code_ != code.CodeSuccess {
		writeRuntimeResponse(c, code_, runtime)
		return
	}
	writeRuntimeResponse(c, code.CodeSuccess, runtime)
}

func RebuildRuntime(c *gin.Context) {
	runtime, code_ := deepagentService.RebuildRuntime(c.Request.Context(), c.GetUint("userID"), c.GetString("userUUID"))
	if code_ != code.CodeSuccess {
		writeRuntimeResponse(c, code_, runtime)
		return
	}
	writeRuntimeResponse(c, code.CodeSuccess, runtime)
}

func parseChatRequest(c *gin.Context) (*ChatRequest, error) {
	req := new(ChatRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		return nil, err
	}
	return req, nil
}

func writeCodeResponse(c *gin.Context, code_ code.Code) {
	c.JSON(http.StatusOK, controller.Response{
		Code: code_,
		Msg:  code_.Msg(),
		Data: gin.H{},
	})
}

func writeRuntimeResponse(c *gin.Context, code_ code.Code, runtime *deepagentService.RuntimeStatus) {
	c.JSON(http.StatusOK, controller.Response{
		Code: code_,
		Msg:  code_.Msg(),
		Data: gin.H{"runtime": runtime},
	})
}

func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
}

func streamSSE(c *gin.Context, events <-chan agentService.StreamEnvelope) {
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case msg, ok := <-events:
			if !ok {
				return false
			}
			c.SSEvent(ginSSEEventName, msg)
			return true
		}
	})
}

func oneShotSSEStream(event agentService.StreamEnvelope) <-chan agentService.StreamEnvelope {
	events := make(chan agentService.StreamEnvelope, 1)
	events <- event
	close(events)
	return events
}

func errorEventStream(code_ code.Code, message string) <-chan agentService.StreamEnvelope {
	return oneShotSSEStream(agentService.NewErrorEvent(code_, message))
}
