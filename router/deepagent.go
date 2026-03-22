package router

import (
	deepagentController "GopherAI/controller/deepagent"

	"github.com/gin-gonic/gin"
)

func SetupDeepAgentRoutes(r *gin.RouterGroup) {
	deepAgent := r.Group("/deep-agent")
	{
		deepAgent.POST("/generate", deepagentController.GenerateHandler)
		deepAgent.POST("/stream", deepagentController.StreamHandler)
		deepAgent.GET("/runtime", deepagentController.GetRuntime)
		deepAgent.POST("/runtime/restart", deepagentController.RestartRuntime)
		deepAgent.POST("/runtime/rebuild", deepagentController.RebuildRuntime)
	}
}
