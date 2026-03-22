package router

import (
	"GopherAI/middleware/jwt"

	"github.com/gin-gonic/gin"
)

func InitRouter() *gin.Engine {
	r := gin.Default()

	api := r.Group("/api/v1")
	{
		// 公开接口（无需认证）
		RegisterUserRouter(api.Group("/user"))
	}

	// 需要 JWT 认证的接口
	auth := api.Group("")
	auth.Use(jwt.Auth())
	{
		// Agent Chat 接口
		// POST /api/v1/agent/generate - 非流式生成
		// POST /api/v1/agent/stream - 流式生成
		// GET /api/v1/agent/:session_id/messages - 获取消息列表
		SetupAgentRoutes(auth)
		SetupDeepAgentRoutes(auth)
		SetupMCPRoutes(auth)

		// 文件相关接口
		FileGroup := auth.Group("/file")
		FileRouter(FileGroup)
	}

	return r
}
