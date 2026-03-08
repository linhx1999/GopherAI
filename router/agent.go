package router

import (
	"github.com/gin-gonic/gin"

	agentController "GopherAI/controller/agent"
	sessionController "GopherAI/controller/session"
	"GopherAI/middleware/jwt"
)

// SetupAgentRoutes 设置 Agent 相关路由
// 统一的 Agent Chat RESTful 接口
func SetupAgentRoutes(r *gin.RouterGroup) {
	// Agent 核心接口（需要认证）
	agent := r.Group("/agent")
	agent.Use(jwt.Auth())
	{
		// POST /api/v1/agent - 发送消息/重新生成（支持流式/非流式）
		agent.POST("", agentController.ChatHandler)

		// GET /api/v1/agent/:session_id/messages - 获取消息列表
		agent.GET("/:session_id/messages", agentController.GetMessages)
	}

	// 工具接口（需要认证）
	tools := r.Group("/tools")
	tools.Use(jwt.Auth())
	{
		// GET /api/v1/tools - 获取可用工具列表
		tools.GET("", agentController.GetTools)
	}

	// 会话接口（需要认证）
	sessions := r.Group("/sessions")
	sessions.Use(jwt.Auth())
	{
		// GET /api/v1/sessions - 获取会话列表
		sessions.GET("", sessionController.GetUserSessionsByUserName)

		// DELETE /api/v1/sessions/:id - 删除会话
		sessions.DELETE("/:id", sessionController.DeleteSession)

		// PUT /api/v1/sessions/:id/title - 更新会话标题
		sessions.PUT("/:id/title", sessionController.UpdateSessionTitle)
	}
}
