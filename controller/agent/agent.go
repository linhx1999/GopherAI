package agent

import (
	"context"
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

// AgentRequest 统一请求结构
type AgentRequest struct {
	SessionID      string   `json:"session_id"`      // 可选，为空则创建新会话
	Message        string   `json:"message"`         // 可选，重新生成时可为空
	RegenerateFrom *int     `json:"regenerate_from"` // 可选，从此索引截断重新生成
	Tools          []string `json:"tools"`           // 可选，工具列表
	ThinkingMode   bool     `json:"thinking_mode"`   // 可选，是否启用思考模型
	Stream         bool     `json:"stream"`          // 是否流式响应，默认 true
}

// AgentHandler 统一 Agent 处理入口
// POST /api/v1/agent
func ChatHandler(c *gin.Context) {
	req := new(AgentRequest)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  "Invalid parameters",
		})
		return
	}

	if req.Stream {
		setSSEHeaders(c)
		streamSSE(c, resolveStreamSource(c.Request.Context(), req, userName))
		return
	}

	handleSyncRequest(c, req, userName)
}

func handleSyncRequest(c *gin.Context, req *AgentRequest, userName string) {
	if req.RegenerateFrom != nil {
		if req.SessionID == "" {
			c.JSON(http.StatusOK, controller.Response{
				Code: code.CodeInvalidParams,
				Msg:  code.CodeInvalidParams.Msg(),
			})
			return
		}
		result, code_ := agentService.Regenerate(c.Request.Context(), userName, req.SessionID, *req.RegenerateFrom, req.Tools, req.ThinkingMode)
		if code_ != code.CodeSuccess {
			c.JSON(http.StatusOK, controller.Response{
				Code: code_,
				Msg:  code_.Msg(),
			})
			return
		}
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeSuccess,
			Msg:  code.CodeSuccess.Msg(),
			Data: []interface{}{gin.H{
				"session_id":    result.SessionID,
				"message_index": result.MessageIndex,
				"message":       result.Message,
			}},
		})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	sessionID := req.SessionID
	if sessionID == "" {
		var code_ code.Code
		sessionID, code_ = agentService.CreateSessionOnly(userName, req.Message)
		if code_ != code.CodeSuccess {
			c.JSON(http.StatusOK, controller.Response{
				Code: code_,
				Msg:  code_.Msg(),
			})
			return
		}
	}

	result, code_ := agentService.GenerateWithContext(c.Request.Context(), userName, sessionID, req.Message, req.Tools, req.ThinkingMode)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, controller.Response{
			Code: code_,
			Msg:  code_.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: []interface{}{gin.H{
			"session_id":    sessionID,
			"message_index": result.MessageIndex,
			"message":       result.Message,
		}},
	})
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
		Data: []interface{}{gin.H{
			"session_id": sessionID,
			"messages":   messages,
			"total":      len(messages),
		}},
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
		Data: []interface{}{gin.H{"tools": toolList}},
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

func resolveStreamSource(ctx context.Context, req *AgentRequest, userName string) <-chan agentService.StreamEvent {
	if req.RegenerateFrom != nil {
		if req.SessionID == "" {
			return errorEventStream(code.CodeInvalidParams, "session_id is required for regenerate")
		}
		handle, code_ := agentService.OpenRegenerateStream(ctx, userName, req.SessionID, *req.RegenerateFrom, req.Tools, req.ThinkingMode)
		if code_ != code.CodeSuccess {
			return errorEventStream(code_, "")
		}
		return handle.Events
	}

	if req.Message == "" {
		return errorEventStream(code.CodeInvalidParams, "message is required")
	}

	handle, code_ := agentService.OpenStreamWithMeta(ctx, userName, req.SessionID, req.Message, req.Tools, req.ThinkingMode)
	if code_ != code.CodeSuccess {
		return errorEventStream(code_, "")
	}
	return handle.Events
}

func errorEventStream(code_ code.Code, message string) <-chan agentService.StreamEvent {
	errorMsg := message
	if errorMsg == "" {
		errorMsg = fmt.Sprintf("Error code: %d", code_)
	}
	return oneShotSSEStream(agentService.NewErrorEvent(errorMsg))
}
