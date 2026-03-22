package router

import (
	"github.com/gin-gonic/gin"

	agentController "GopherAI/controller/agent"
	sessionController "GopherAI/controller/session"
)

// SetupAgentRoutes 设置 Agent 相关路由
func SetupAgentRoutes(r *gin.RouterGroup) {
	agent := r.Group("/agent")
	{
		// POST /api/v1/agent/generate - 非流式生成
		agent.POST("/generate", agentController.GenerateHandler)

		// POST /api/v1/agent/stream - 流式生成
		agent.POST("/stream", agentController.StreamHandler)

		// GET /api/v1/agent/:session_id/messages - 获取消息列表
		agent.GET("/:session_id/messages", agentController.GetMessages)
	}

	tools := r.Group("/tools")
	{
		// GET /api/v1/tools - 获取可用工具列表
		tools.GET("", agentController.GetTools)
	}

	sessions := r.Group("/sessions")
	{
		// POST /api/v1/sessions - 创建会话
		sessions.POST("", sessionController.CreateSession)

		// GET /api/v1/sessions - 获取会话列表
		sessions.GET("", sessionController.GetUserSessionsByUserName)

		// DELETE /api/v1/sessions/:session_id - 删除会话
		sessions.DELETE("/:session_id", sessionController.DeleteSession)

		// PUT /api/v1/sessions/:session_id/title - 更新会话标题
		sessions.PUT("/:session_id/title", sessionController.UpdateSessionTitle)
	}
}
