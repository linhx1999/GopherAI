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
	"GopherAI/model"
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

// MessageItem 消息项
type MessageItem struct {
	Index     int              `json:"index"`
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []model.ToolCall `json:"tool_calls,omitempty"`
	CreatedAt string           `json:"created_at"`
}

// AgentHandler 统一 Agent 处理入口
// POST /api/v1/agent
// 支持两种场景：
// 1. 正常对话：传入 message，可选传入 session_id 和 tools
// 2. 重新生成：传入 session_id 和 regenerate_from
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

	// 默认流式响应
	stream := req.Stream
	if stream {
		setSSEHeaders(c)
		events := make(chan agentService.SSEEvent)
		go handleStreamRequest(c, events, req, userName)

		c.Stream(func(w io.Writer) bool {
			select {
			// 监听客户端是否主动断开连接
			case <-c.Request.Context().Done():
				return false

			// 接收大模型吐出的数据
			case msg, ok := <-events:
				if !ok {
					// 通道关闭，说明大模型生成结束
					// 按照 SSE 规范，通常会发送一个 [DONE] 标识，前端可以据此结束接收
					c.SSEvent(ginSSEEventName, "[DONE]")
					return false
				}

				// c.SSEvent 会自动帮你把数据包装成 SSE 的格式: "data: {msg}\n\n"
				// 第一个参数是事件名（通常默认留空或者写 message），第二个参数是数据本身
				c.SSEvent(ginSSEEventName, msg)
				return true
			}
		})
	} else {
		handleSyncRequest(c, req, userName)
	}
}

// handleStreamRequest 处理流式请求
func handleStreamRequest(c *gin.Context, events chan<- agentService.SSEEvent, req *AgentRequest, userName string) {
	defer close(events)

	source := resolveStreamSource(c, req, userName)
	if source == nil {
		return
	}
	forwardSSEEvents(c.Request.Context(), events, source)
}

// handleSyncRequest 处理同步请求
func handleSyncRequest(c *gin.Context, req *AgentRequest, userName string) {
	// 重新生成场景
	if req.RegenerateFrom != nil {
		if req.SessionID == "" {
			c.JSON(http.StatusOK, controller.Response{
				Code: code.CodeInvalidParams,
				Msg:  code.CodeInvalidParams.Msg(),
			})
			return
		}
		result, code_ := agentService.Regenerate(c, userName, req.SessionID, *req.RegenerateFrom, req.Tools, req.ThinkingMode)
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
				"session_id":        result.SessionID,
				"message_index":     result.MessageIndex,
				"role":              result.Role,
				"content":           result.Content,
				"reasoning_content": result.ReasoningContent,
				"tool_calls":        result.ToolCalls,
			}},
		})
		return
	}

	// 验证消息
	if req.Message == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	// 创建会话（如果需要）
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

	// 正常对话
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
			"session_id":        sessionID,
			"message_index":     result.MessageIndex,
			"role":              result.Role,
			"content":           result.Content,
			"reasoning_content": result.ReasoningContent,
			"tool_calls":        result.ToolCalls,
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

	// 转换为响应格式
	items := make([]interface{}, 0, len(messages))
	for _, msg := range messages {
		item := MessageItem{
			Index:     msg.Index,
			Role:      msg.Role,
			Content:   msg.Content,
			CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if msg.ToolCalls != nil {
			item.ToolCalls = msg.GetToolCalls()
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: []interface{}{gin.H{
			"session_id": sessionID,
			"messages":   items,
			"total":      len(items),
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

func singleSSEvent(event agentService.SSEEvent) <-chan agentService.SSEEvent {
	events := make(chan agentService.SSEEvent, 1)
	events <- event
	close(events)
	return events
}

func resolveStreamSource(c *gin.Context, req *AgentRequest, userName string) <-chan agentService.SSEEvent {
	if req.RegenerateFrom != nil {
		if req.SessionID == "" {
			return errorEventStream(code.CodeInvalidParams, "session_id is required for regenerate")
		}
		handle, code_ := agentService.OpenRegenerateStream(c, userName, req.SessionID, *req.RegenerateFrom, req.Tools, req.ThinkingMode)
		if code_ != code.CodeSuccess {
			return errorEventStream(code_, "")
		}
		return handle.Events
	}

	if req.Message == "" {
		return errorEventStream(code.CodeInvalidParams, "message is required")
	}

	handle, code_ := agentService.OpenStreamWithMeta(c, userName, req.SessionID, req.Message, req.Tools, req.ThinkingMode)
	if code_ != code.CodeSuccess {
		return errorEventStream(code_, "")
	}
	return handle.Events
}

func forwardSSEEvents(ctx context.Context, dst chan<- agentService.SSEEvent, src <-chan agentService.SSEEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-src:
			if !ok {
				return
			}
			select {
			case <-ctx.Done():
				return
			case dst <- event:
			}
		}
	}
}

func errorEventStream(code_ code.Code, message string) <-chan agentService.SSEEvent {
	errorMsg := message
	if errorMsg == "" {
		errorMsg = fmt.Sprintf("Error code: %d", code_)
	}
	return singleSSEvent(agentService.NewErrorEvent(errorMsg))
}
