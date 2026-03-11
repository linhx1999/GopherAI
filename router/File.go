package router

import (
	"GopherAI/controller/file"

	"github.com/gin-gonic/gin"
)

func FileRouter(r *gin.RouterGroup) {
	r.POST("/upload", file.UploadRagFile)
	r.GET("/list", file.GetFileList)
	r.GET("/url/:file_id", file.GetFileURL)
	r.GET("/download/:file_id", file.DownloadFile)
	r.DELETE("/:file_id", file.DeleteFile)
	r.POST("/index/:file_id", file.IndexFile)
	r.DELETE("/index/:file_id", file.DeleteFileIndex)
}
