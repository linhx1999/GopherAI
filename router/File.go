package router

import (
	"GopherAI/controller/file"

	"github.com/gin-gonic/gin"
)

func FileRouter(r *gin.RouterGroup) {
	r.POST("/upload", file.UploadRagFile)
	r.GET("/list", file.GetFileList)
	r.GET("/url/:id", file.GetFileURL)
	r.GET("/download/:id", file.DownloadFile)
	r.DELETE("/:id", file.DeleteFile)
	r.POST("/index/:id", file.IndexFile)
	r.DELETE("/index/:id", file.DeleteFileIndex)
}