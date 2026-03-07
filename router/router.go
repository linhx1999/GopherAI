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
		// Agent Chat 接口 - 统一的 RESTful 接口
		// POST /api/v1/agent - 发送消息/重新生成
		// GET /api/v1/agent/:session_id/messages - 获取消息列表
		SetupAgentRoutes(auth)

		// 图片相关接口
		ImageGroup := auth.Group("/image")
		ImageRouter(ImageGroup)

		// 文件相关接口
		FileGroup := auth.Group("/file")
		FileRouter(FileGroup)
	}

	return r
}