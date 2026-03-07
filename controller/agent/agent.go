package agent

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"GopherAI/common/code"
	"GopherAI/common/agent/tools"
	"GopherAI/controller"
	agentService "GopherAI/service/agent"
	"GopherAI/model"
)

// AgentRequest 统一请求结构
type AgentRequest struct {
	SessionID      string   `json:"session_id"`       // 可选，为空则创建新会话
	Message        string   `json:"message"`          // 可选，重新生成时可为空
	RegenerateFrom *int     `json:"regenerate_from"`  // 可选，从此索引截断重新生成
	Tools          []string `json:"tools"`            // 可选，工具列表
	Stream         bool     `json:"stream"`           // 是否流式响应，默认 true
}

// AgentResponse 统一响应结构
type AgentResponse struct {
	SessionID    string           `json:"session_id"`
	MessageIndex int              `json:"message_index"`
	Role         string           `json:"role"`
	Content      string           `json:"content"`
	ToolCalls    []model.ToolCall `json:"tool_calls,omitempty"`
	controller.Response
}

// MessageItem 消息项
type MessageItem struct {
	Index     int              `json:"index"`
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []model.ToolCall `json:"tool_calls,omitempty"`
	CreatedAt string           `json:"created_at"`
}

// MessagesResponse 消息列表响应
type MessagesResponse struct {
	SessionID string        `json:"session_id"`
	Messages  []MessageItem `json:"messages"`
	Total     int           `json:"total"`
}

// ToolsResponse 工具列表响应
type ToolsResponse struct {
	Tools []tools.ToolInfo `json:"tools"`
}

// AgentHandler 统一 Agent 处理入口
// POST /api/v1/agent
// 支持两种场景：
// 1. 正常对话：传入 message，可选传入 session_id 和 tools
// 2. 重新生成：传入 session_id 和 regenerate_from
func AgentHandler(c *gin.Context) {
	req := new(AgentRequest)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": code.CodeInvalidParams,
			"msg":  "Invalid parameters",
		})
		return
	}

	// 默认流式响应
	stream := req.Stream
	if stream {
		handleStreamRequest(c, req, userName)
	} else {
		handleSyncRequest(c, req, userName)
	}
}

// handleStreamRequest 处理流式请求
func handleStreamRequest(c *gin.Context, req *AgentRequest, userName string) {
	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 重新生成场景
	if req.RegenerateFrom != nil {
		if req.SessionID == "" {
			sendSSEError(c.Writer, code.CodeInvalidParams, "session_id is required for regenerate")
			return
		}
		result, code_ := agentService.Regenerate(c.Request.Context(), userName, req.SessionID, *req.RegenerateFrom, req.Tools, c.Writer, true)
		if code_ != code.CodeSuccess {
			sendSSEError(c.Writer, code_, "")
		}
		_ = result
		return
	}

	// 验证消息
	if req.Message == "" {
		sendSSEError(c.Writer, code.CodeInvalidParams, "message is required")
		return
	}

	// 正常对话场景
	result, code_ := agentService.StreamWithMeta(userName, req.SessionID, req.Message, req.Tools, c.Writer)
	if code_ != code.CodeSuccess {
		sendSSEError(c.Writer, code_, "")
		return
	}
	_ = result
}

// handleSyncRequest 处理同步请求
func handleSyncRequest(c *gin.Context, req *AgentRequest, userName string) {
	res := new(AgentResponse)

	// 重新生成场景
	if req.RegenerateFrom != nil {
		if req.SessionID == "" {
			c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
			return
		}
		result, code_ := agentService.Regenerate(c.Request.Context(), userName, req.SessionID, *req.RegenerateFrom, req.Tools, nil, false)
		if code_ != code.CodeSuccess {
			c.JSON(http.StatusOK, res.CodeOf(code_))
			return
		}
		res.Success()
		res.SessionID = result.SessionID
		res.MessageIndex = result.MessageIndex
		res.Role = result.Role
		res.Content = result.Content
		res.ToolCalls = result.ToolCalls
		c.JSON(http.StatusOK, res)
		return
	}

	// 验证消息
	if req.Message == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	// 创建会话（如果需要）
	sessionID := req.SessionID
	if sessionID == "" {
		var code_ code.Code
		sessionID, code_ = agentService.CreateSessionOnly(userName, req.Message)
		if code_ != code.CodeSuccess {
			c.JSON(http.StatusOK, res.CodeOf(code_))
			return
		}
	}

	// 正常对话
	result, code_ := agentService.GenerateWithContext(c.Request.Context(), userName, sessionID, req.Message, req.Tools)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.SessionID = sessionID
	res.MessageIndex = result.MessageIndex
	res.Role = result.Role
	res.Content = result.Content
	res.ToolCalls = result.ToolCalls
	c.JSON(http.StatusOK, res)
}

// GetMessages 获取消息列表
// GET /api/v1/agent/:session_id/messages
func GetMessages(c *gin.Context) {
	sessionID := c.Param("session_id")
	userName := c.GetString("userName")

	messages, err := agentService.GetMessages(sessionID, userName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": code.CodeServerBusy,
			"msg":  "Failed to get messages",
		})
		return
	}

	// 转换为响应格式
	items := make([]MessageItem, 0, len(messages))
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

	c.JSON(http.StatusOK, gin.H{
		"code": code.CodeSuccess,
		"data": MessagesResponse{
			SessionID: sessionID,
			Messages:  items,
			Total:     len(items),
		},
	})
}

// GetTools 获取可用工具列表
// GET /api/v1/tools
func GetTools(c *gin.Context) {
	registry := tools.GetToolRegistry()
	toolList := registry.ListAvailableTools(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"code": code.CodeSuccess,
		"data": ToolsResponse{
			Tools: toolList,
		},
	})
}

// sendSSEError 发送 SSE 错误
func sendSSEError(writer http.ResponseWriter, code_ code.Code, message string) {
	errorMsg := message
	if errorMsg == "" {
		errorMsg = fmt.Sprintf("Error code: %d", code_)
	}
	writer.Write([]byte(fmt.Sprintf("data: {\"type\":\"error\",\"message\":\"%s\"}\n\n", errorMsg)))
	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}
}