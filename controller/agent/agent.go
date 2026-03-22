package agent

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"GopherAI/common/agent/tools"
	"GopherAI/common/code"
	"GopherAI/controller"
	agentService "GopherAI/service/agent"
)

const ginSSEEventName = "message"

// ChatRequest 表示生成类接口的共享请求体。
type ChatRequest struct {
	SessionID    string   `json:"session_id"`                 // 流式接口必填，非流式接口可选
	Message      string   `json:"message" binding:"required"` // 必填，用户消息内容
	Tools        []string `json:"tools"`                      // 可选，工具 API 调用名列表
	ThinkingMode bool     `json:"thinking_mode"`              // 可选，是否启用思考模型
}

// GenerateHandler 处理非流式生成请求。
// POST /api/v1/agent/generate
func GenerateHandler(c *gin.Context) {
	req, err := parseChatRequest(c)
	if err != nil {
		writeCodeResponse(c, code.CodeInvalidParams)
		return
	}

	result, code_ := agentService.Generate(
		c.Request.Context(),
		c.GetUint("userID"),
		req.SessionID,
		req.Message,
		req.Tools,
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

// StreamHandler 处理 SSE 流式生成请求。
// POST /api/v1/agent/stream
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

	stream, code_ := agentService.Stream(
		c.Request.Context(),
		c.GetUint("userID"),
		req.SessionID,
		req.Message,
		req.Tools,
		req.ThinkingMode,
	)
	if code_ != code.CodeSuccess {
		streamSSE(c, errorEventStream(code_, ""))
		return
	}

	streamSSE(c, stream.Events)
}

// GetMessages 获取消息列表
// GET /api/v1/agent/:session_id/messages
func GetMessages(c *gin.Context) {
	sessionID := c.Param("session_id")
	userID := c.GetUint("userID")

	messages, code_ := agentService.ListHistoryMessages(sessionID, userID)
	if code_ != code.CodeSuccess {
		writeCodeResponse(c, code_)
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: gin.H{
			"session_id": sessionID,
			"messages":   messages,
			"total":      len(messages),
		},
	})
}

// GetTools 获取可用工具列表
// GET /api/v1/tools
func GetTools(c *gin.Context) {
	toolList := tools.ListAvailableTools()

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: gin.H{"tools": toolList},
	})
}

func parseChatRequest(c *gin.Context) (*ChatRequest, error) {
	req := new(ChatRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		return nil, err
	}
	return req, nil
}

func writeCodeResponse(c *gin.Context, code code.Code) {
	c.JSON(http.StatusOK, controller.Response{
		Code: code,
		Msg:  code.Msg(),
		Data: gin.H{},
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
