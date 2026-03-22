package router

import (
	mcpController "GopherAI/controller/mcp"

	"github.com/gin-gonic/gin"
)

func SetupMCPRoutes(r *gin.RouterGroup) {
	mcp := r.Group("/mcp/servers")
	{
		mcp.GET("", mcpController.ListServers)
		mcp.GET("/:server_id", mcpController.GetServer)
		mcp.POST("", mcpController.CreateServer)
		mcp.PUT("/:server_id", mcpController.UpdateServer)
		mcp.DELETE("/:server_id", mcpController.DeleteServer)
		mcp.POST("/:server_id/test", mcpController.TestServer)
	}
}
