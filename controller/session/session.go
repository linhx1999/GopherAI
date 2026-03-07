package session

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"GopherAI/common/code"
	"GopherAI/controller"
	sessionService "GopherAI/service/session"
)

type (
	// ChatSendRequest 聊天发送请求
	ChatSendRequest struct {
		Content   string `json:"content" binding:"required"` // 用户内容
		SessionID string `json:"session_id"`                 // 会话ID（可选，为空则创建新会话）
	}

	// ChatHistoryRequest 聊天历史请求
	ChatHistoryRequest struct {
		SessionID string `json:"session_id" binding:"required"`
	}

	// UpdateSessionTitleRequest 更新会话标题请求
	UpdateSessionTitleRequest struct {
		Title string `json:"title" binding:"required"`
	}
)

func GetUserSessionsByUserName(c *gin.Context) {
	userName := c.GetString("userName")

	userSessions, err := sessionService.GetUserSessionsByUserName(userName)
	if err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeServerBusy,
			Msg:  code.CodeServerBusy.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: []interface{}{gin.H{"sessions": userSessions}},
	})
}

func CreateSessionAndSendMessage(c *gin.Context) {
	req := new(ChatSendRequest)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	sessionID, aiContent, code_ := sessionService.CreateSessionAndSendMessage(userName, req.Content)
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
			"content":    aiContent,
			"session_id": sessionID,
		}},
	})
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
	sessionID, code_ := sessionService.CreateStreamSessionOnly(userName, req.Content)
	if code_ != code.CodeSuccess {
		c.SSEvent("error", gin.H{"message": "Failed to create session"})
		return
	}

	c.Writer.WriteString(fmt.Sprintf("data: {\"session_id\": \"%s\"}\n\n", sessionID))
	c.Writer.Flush()

	// 流式响应
	code_ = sessionService.StreamMessageToExistingSession(userName, sessionID, req.Content, c.Writer)
	if code_ != code.CodeSuccess {
		c.SSEvent("error", gin.H{"message": "Failed to send message"})
		return
	}
}

func ChatSend(c *gin.Context) {
	req := new(ChatSendRequest)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	if req.SessionID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	aiContent, code_ := sessionService.ChatSend(userName, req.SessionID, req.Content)
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
			"content":    aiContent,
			"session_id": req.SessionID,
		}},
	})
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

	code_ := sessionService.ChatStreamSend(userName, req.SessionID, req.Content, c.Writer)
	if code_ != code.CodeSuccess {
		c.SSEvent("error", gin.H{"message": "Failed to send message"})
		return
	}
}

func ChatHistory(c *gin.Context) {
	req := new(ChatHistoryRequest)
	userName := c.GetString("userName")

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	messages, code_ := sessionService.GetChatHistory(userName, req.SessionID)
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
		Data: []interface{}{gin.H{"messages": messages}},
	})
}

func DeleteSession(c *gin.Context) {
	userName := c.GetString("userName")
	sessionID := c.Param("id")

	if sessionID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	code_ := sessionService.DeleteSession(userName, sessionID)
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
	})
}

func UpdateSessionTitle(c *gin.Context) {
	userName := c.GetString("userName")
	sessionID := c.Param("sessionId")

	if sessionID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	req := new(UpdateSessionTitleRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	code_ := sessionService.UpdateSessionTitle(userName, sessionID, req.Title)
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
	})
}