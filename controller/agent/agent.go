package agent

import (
	"errors"
	"fmt"
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
	SessionID    string   `json:"session_id"`    // 可选，为空则创建新会话
	Message      string   `json:"message"`       // 必填，用户消息内容
	Tools        []string `json:"tools"`         // 可选，工具列表
	ThinkingMode bool     `json:"thinking_mode"` // 可选，是否启用思考模型
}

// GenerateHandler 处理非流式生成请求。
// POST /api/v1/agent/generate
func GenerateHandler(c *gin.Context) {
	req, err := parseChatRequest(c)
	if err != nil {
		writeCodeResponse(c, code.CodeInvalidParams)
		return
	}

	result, code_ := agentService.Generate(c.Request.Context(), agentService.GenerateRequest{
		UserName:     c.GetString("userName"),
		SessionID:    req.SessionID,
		UserMessage:  req.Message,
		ToolNames:    req.Tools,
		ThinkingMode: req.ThinkingMode,
	})
	if code_ != code.CodeSuccess {
		writeCodeResponse(c, code_)
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
	if err != nil {
		setSSEHeaders(c)
		streamSSE(c, errorEventStream(code.CodeInvalidParams, err.Error()))
		return
	}

	stream, code_ := agentService.Stream(c.Request.Context(), agentService.GenerateRequest{
		UserName:     c.GetString("userName"),
		SessionID:    req.SessionID,
		UserMessage:  req.Message,
		ToolNames:    req.Tools,
		ThinkingMode: req.ThinkingMode,
	})
	if code_ != code.CodeSuccess {
		setSSEHeaders(c)
		streamSSE(c, errorEventStream(code_, ""))
		return
	}

	setSSEHeaders(c)
	streamSSE(c, stream.Events)
}

// GetMessages 获取消息列表
// GET /api/v1/agent/:session_id/messages
func GetMessages(c *gin.Context) {
	sessionID := c.Param("session_id")
	userName := c.GetString("userName")

	messages, err := agentService.GetMessages(sessionID, userName)
	if err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeServerBusy,
			Msg:  "Failed to get messages",
		})
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
	registry := tools.GetToolRegistry()
	toolList := registry.ListAvailableTools(c.Request.Context())

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: gin.H{"tools": toolList},
	})
}

func parseChatRequest(c *gin.Context) (*ChatRequest, error) {
	req := new(ChatRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		return nil, errors.New("invalid parameters")
	}
	if req.Message == "" {
		return nil, errors.New("message is required")
	}
	return req, nil
}

func writeCodeResponse(c *gin.Context, code_ code.Code) {
	c.JSON(http.StatusOK, controller.Response{
		Code: code_,
		Msg:  code_.Msg(),
	})
}

func setSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
}

func streamSSE(c *gin.Context, events <-chan agentService.StreamEvent) {
	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case msg, ok := <-events:
			if !ok {
				c.SSEvent(ginSSEEventName, "[DONE]")
				return false
			}

			switch {
			case msg.Meta != nil:
				c.SSEvent(ginSSEEventName, msg.Meta)
			case msg.Error != nil:
				c.SSEvent(ginSSEEventName, msg.Error)
			case msg.Message != nil:
				c.SSEvent(ginSSEEventName, msg.Message)
			}
			return true
		}
	})
}

func oneShotSSEStream(event agentService.StreamEvent) <-chan agentService.StreamEvent {
	events := make(chan agentService.StreamEvent, 1)
	events <- event
	close(events)
	return events
}

func errorEventStream(code_ code.Code, message string) <-chan agentService.StreamEvent {
	errorMsg := message
	if errorMsg == "" {
		errorMsg = fmt.Sprintf("Error code: %d", code_)
	}
	return oneShotSSEStream(agentService.NewErrorEvent(errorMsg))
}
