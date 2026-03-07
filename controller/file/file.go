package file

import (
	"GopherAI/common/code"
	"GopherAI/controller"
	ragService "GopherAI/service/rag"
	fileService "GopherAI/service/file"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FileInfo struct {
	ID            uint   `json:"id"`
	FileName      string `json:"file_name"`
	FileSize      string `json:"file_size"`
	FileSizeBytes int64  `json:"file_size_bytes"`
	FileType      string `json:"file_type"`
	CreatedAt     string `json:"created_at"`
	ObjectName    string `json:"object_name,omitempty"`
	IndexStatus   string `json:"index_status"`  // 索引状态：pending/indexing/indexed/failed
	IndexMessage  string `json:"index_message"` // 索引状态消息
}

// UploadRagFile 上传 RAG 文件
func UploadRagFile(c *gin.Context) {
	uploadedFile, err := c.FormFile("file")
	if err != nil {
		log.Println("FormFile fail ", err)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	username := c.GetString("userName")
	if username == "" {
		log.Println("Username not found in context")
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidToken,
			Msg:  code.CodeInvalidToken.Msg(),
		})
		return
	}

	fileRecord, err := fileService.UploadRagFile(username, uploadedFile)
	if err != nil {
		log.Println("UploadFile fail ", err)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeServerBusy,
			Msg:  code.CodeServerBusy.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: []interface{}{FileInfo{
			ID:            fileRecord.ID,
			FileName:      fileRecord.FileName,
			FileSize:      fileService.FormatFileSize(fileRecord.FileSize),
			FileSizeBytes: fileRecord.FileSize,
			FileType:      fileRecord.FileType,
			CreatedAt:     fileService.FormatTime(fileRecord.CreatedAt),
			ObjectName:    fileRecord.ObjectName,
		}},
	})
}

// GetFileList 获取用户文件列表
func GetFileList(c *gin.Context) {
	username := c.GetString("userName")
	if username == "" {
		log.Println("Username not found in context")
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidToken,
			Msg:  code.CodeInvalidToken.Msg(),
		})
		return
	}

	files, err := fileService.GetFileList(username)
	if err != nil {
		log.Println("GetFileList fail ", err)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeServerBusy,
			Msg:  code.CodeServerBusy.Msg(),
		})
		return
	}

	fileInfos := make([]interface{}, 0, len(files))
	for _, f := range files {
		fileInfos = append(fileInfos, FileInfo{
			ID:            f.ID,
			FileName:      f.FileName,
			FileSize:      fileService.FormatFileSize(f.FileSize),
			FileSizeBytes: f.FileSize,
			FileType:      f.FileType,
			CreatedAt:     fileService.FormatTime(f.CreatedAt),
			IndexStatus:   f.IndexStatus,
			IndexMessage:  f.IndexMessage,
		})
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: fileInfos,
	})
}

// DeleteFile 删除文件
func DeleteFile(c *gin.Context) {
	username := c.GetString("userName")
	if username == "" {
		log.Println("Username not found in context")
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidToken,
			Msg:  code.CodeInvalidToken.Msg(),
		})
		return
	}

	// 从 URL 参数获取文件 ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		log.Println("Invalid file ID: ", idStr)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	if err := fileService.DeleteFile(uint(id), username); err != nil {
		log.Println("DeleteFile fail ", err)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeServerBusy,
			Msg:  code.CodeServerBusy.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
	})
}

// GetFileURL 获取文件访问 URL
func GetFileURL(c *gin.Context) {
	username := c.GetString("userName")
	if username == "" {
		log.Println("Username not found in context")
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidToken,
			Msg:  code.CodeInvalidToken.Msg(),
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		log.Println("Invalid file ID: ", idStr)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	url, err := fileService.GetFileURL(uint(id), username)
	if err != nil {
		log.Println("GetFileURL fail ", err)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeServerBusy,
			Msg:  code.CodeServerBusy.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: []interface{}{gin.H{"url": url}},
	})
}

// DownloadFile 下载文件
func DownloadFile(c *gin.Context) {
	username := c.GetString("userName")
	if username == "" {
		log.Println("Username not found in context")
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidToken,
			Msg:  code.CodeInvalidToken.Msg(),
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		log.Println("Invalid file ID: ", idStr)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	reader, fileRecord, err := fileService.DownloadFileContent(uint(id), username)
	if err != nil {
		log.Println("DownloadFile fail ", err)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeServerBusy,
			Msg:  code.CodeServerBusy.Msg(),
		})
		return
	}
	defer reader.Close()

	// 设置响应头
	contentType := "text/plain"
	if fileRecord.FileType == ".md" {
		contentType = "text/markdown"
	}
	c.Header("Content-Disposition", "attachment; filename="+fileRecord.FileName)
	c.DataFromReader(http.StatusOK, fileRecord.FileSize, contentType, reader, nil)
}

// IndexFile 手动触发文件索引
func IndexFile(c *gin.Context) {
	username := c.GetString("userName")
	if username == "" {
		log.Println("Username not found in context")
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidToken,
			Msg:  code.CodeInvalidToken.Msg(),
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		log.Println("Invalid file ID: ", idStr)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	// 创建 RAG 服务实例
	ragSvc := &ragService.RAGService{}

	// 异步执行索引（避免阻塞请求）
	go func() {
		if err := ragSvc.IndexFile(uint(id)); err != nil {
			log.Printf("Failed to index file %d: %v", id, err)
		}
	}()

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
	})
}

// DeleteFileIndex 删除文件索引
func DeleteFileIndex(c *gin.Context) {
	username := c.GetString("userName")
	if username == "" {
		log.Println("Username not found in context")
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidToken,
			Msg:  code.CodeInvalidToken.Msg(),
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		log.Println("Invalid file ID: ", idStr)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	// 创建 RAG 服务实例
	ragSvc := &ragService.RAGService{}

	if err := ragSvc.DeleteIndex(uint(id)); err != nil {
		log.Println("DeleteFileIndex fail: ", err)
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeServerBusy,
			Msg:  code.CodeServerBusy.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
	})
}
