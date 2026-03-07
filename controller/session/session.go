package session

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"GopherAI/common/code"
	"GopherAI/controller"
	"GopherAI/model"
	"GopherAI/service/session"
)

type (
	GetUserSessionsResponse struct {
		controller.Response
		Sessions []model.SessionInfo `json:"sessions,omitempty"`
	}

	// ChatSendRequest 聊天发送请求
	ChatSendRequest struct {
		Content   string `json:"content" binding:"required"`   // 用户内容
		SessionID string `json:"session_id"`                   // 会话ID（可选，为空则创建新会话）
	}

	// ChatSendResponse 聊天发送响应
	ChatSendResponse struct {
		Content   string `json:"content,omitempty"` // AI 响应内容
		SessionID string `json:"session_id,omitempty"`
		controller.Response
	}

	// ChatHistoryRequest 聊天历史请求
	ChatHistoryRequest struct {
		SessionID string `json:"session_id" binding:"required"`
	}

	// ChatHistoryResponse 聊天历史响应
	ChatHistoryResponse struct {
		Messages []*model.Message `json:"messages"`
		controller.Response
	}

	// UpdateSessionTitleRequest 更新会话标题请求
	UpdateSessionTitleRequest struct {
		Title string `json:"title" binding:"required"`
	}
)

func GetUserSessionsByUserName(c *gin.Context) {
	res := new(GetUserSessionsResponse)
	userName := c.GetString("userName")

	userSessions, err := session.GetUserSessionsByUserName(userName)
	if err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.Sessions = userSessions
	c.JSON(http.StatusOK, res)
}

func CreateSessionAndSendMessage(c *gin.Context) {
	req := new(ChatSendRequest)
	res := new(ChatSendResponse)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	sessionID, aiContent, code_ := session.CreateSessionAndSendMessage(userName, req.Content)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.Content = aiContent
	res.SessionID = sessionID
	c.JSON(http.StatusOK, res)
}

func CreateStreamSessionAndSendMessage(c *gin.Context) {
	req := new(ChatSendRequest)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "Invalid parameters"})
		return
	}

	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no")

	// 创建会话并发送 session_id
	sessionID, code_ := session.CreateStreamSessionOnly(userName, req.Content)
	if code_ != code.CodeSuccess {
		c.SSEvent("error", gin.H{"message": "Failed to create session"})
		return
	}

	c.Writer.WriteString(fmt.Sprintf("data: {\"session_id\": \"%s\"}\n\n", sessionID))
	c.Writer.Flush()

	// 流式响应
	code_ = session.StreamMessageToExistingSession(userName, sessionID, req.Content, c.Writer)
	if code_ != code.CodeSuccess {
		c.SSEvent("error", gin.H{"message": "Failed to send message"})
		return
	}
}

func ChatSend(c *gin.Context) {
	req := new(ChatSendRequest)
	res := new(ChatSendResponse)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	if req.SessionID == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	aiContent, code_ := session.ChatSend(userName, req.SessionID, req.Content)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.Content = aiContent
	res.SessionID = req.SessionID
	c.JSON(http.StatusOK, res)
}

func ChatStreamSend(c *gin.Context) {
	req := new(ChatSendRequest)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "Invalid parameters"})
		return
	}

	if req.SessionID == "" {
		c.JSON(http.StatusOK, gin.H{"error": "session_id is required"})
		return
	}

	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("X-Accel-Buffering", "no")

	code_ := session.ChatStreamSend(userName, req.SessionID, req.Content, c.Writer)
	if code_ != code.CodeSuccess {
		c.SSEvent("error", gin.H{"message": "Failed to send message"})
		return
	}
}

func ChatHistory(c *gin.Context) {
	req := new(ChatHistoryRequest)
	res := new(ChatHistoryResponse)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	messages, code_ := session.GetChatHistory(userName, req.SessionID)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.Messages = messages
	c.JSON(http.StatusOK, res)
}

func DeleteSession(c *gin.Context) {
	res := new(controller.Response)
	userName := c.GetString("userName")
	sessionID := c.Param("sessionId")

	if sessionID == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	code_ := session.DeleteSession(userName, sessionID)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	c.JSON(http.StatusOK, res)
}

func UpdateSessionTitle(c *gin.Context) {
	res := new(controller.Response)
	userName := c.GetString("userName")
	sessionID := c.Param("sessionId")

	if sessionID == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	req := new(UpdateSessionTitleRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	code_ := session.UpdateSessionTitle(userName, sessionID, req.Title)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	c.JSON(http.StatusOK, res)
}