package file

import (
	"GopherAI/common/minio"
	"GopherAI/common/rag"
	"GopherAI/config"
	"GopherAI/dao/file"
	"GopherAI/model"
	"GopherAI/utils"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
)

// UploadRagFile 上传 RAG 文件到 MinIO 并创建索引
func UploadRagFile(userRefID uint, username string, fileHeader *multipart.FileHeader) (*model.File, error) {
	// 校验文件类型和文件名
	if err := utils.ValidateFile(fileHeader); err != nil {
		log.Printf("File validation failed: %v", err)
		return nil, err
	}

	ctx := context.Background()

	// 生成 UUID 作为对象名
	uuid := utils.GenerateUUID()
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	objectName := fmt.Sprintf("%s/%s%s", username, uuid, ext)

	// 打开上传的文件
	src, err := fileHeader.Open()
	if err != nil {
		log.Printf("Failed to open uploaded file: %v", err)
		return nil, err
	}
	defer src.Close()

	// 确定内容类型
	contentType := "text/plain"
	if ext == ".md" {
		contentType = "text/markdown"
	}

	// 上传到 MinIO
	if err := minio.UploadFile(ctx, objectName, src, fileHeader.Size, contentType); err != nil {
		log.Printf("Failed to upload file to MinIO: %v", err)
		return nil, err
	}

	// 创建文件元数据记录
	fileRecord := &model.File{
		FileID:     utils.GenerateUUID(),
		UserRefID:  userRefID,
		FileName:   fileHeader.Filename,
		ObjectName: objectName,
		FileSize:   fileHeader.Size,
		FileType:   ext,
		Bucket:     config.GetConfig().MinioConfig.Bucket,
	}

	if err := file.Create(fileRecord); err != nil {
		log.Printf("Failed to create file record: %v", err)
		// 回滚：删除 MinIO 中的文件
		minio.DeleteFile(ctx, objectName)
		return nil, err
	}

	log.Printf("File uploaded successfully: %s", objectName)

	// 从 MinIO 下载文件内容用于 RAG 索引
	obj, err := minio.DownloadFile(ctx, objectName)
	if err != nil {
		log.Printf("Failed to download file for indexing: %v", err)
		return fileRecord, nil // 文件已上传成功，只是索引失败
	}
	defer obj.Close()

	// 读取文件内容
	content, err := io.ReadAll(obj)
	if err != nil {
		log.Printf("Failed to read file content: %v", err)
		return fileRecord, nil
	}

	// 创建 RAG 索引（使用文件 ID 作为索引标识）
	indexer, err := rag.NewRAGIndexer(ctx, fileRecord.ID, content, ext, config.GetConfig().RagModelConfig.RagEmbeddingModel)
	if err != nil {
		log.Printf("Failed to create RAG indexer: %v", err)
		return fileRecord, nil
	}

	// 创建向量索引
	if err := indexer.IndexFileContent(ctx); err != nil {
		log.Printf("Failed to index file: %v", err)
		// 不删除文件，仅记录错误
	}

	return fileRecord, nil
}

// GetFileList 获取用户文件列表
func GetFileList(userRefID uint) ([]model.File, error) {
	return file.GetByUserRefID(userRefID)
}

// DeleteFile 删除文件（MinIO 对象 + 数据库记录 + Redis 索引）
func DeleteFile(fileID string, userRefID uint) error {
	ctx := context.Background()

	// 获取文件记录（同时校验用户权限）
	fileRecord, err := file.GetByFileIDAndUserRefID(fileID, userRefID)
	if err != nil {
		return err
	}
	if fileRecord == nil {
		return fmt.Errorf("文件不存在或无权限删除")
	}

	// 删除 MinIO 对象
	if err := minio.DeleteFile(ctx, fileRecord.ObjectName); err != nil {
		log.Printf("Failed to delete file from MinIO: %v", err)
		// 继续执行，即使 MinIO 删除失败也删除数据库记录
	}

	// 删除 Redis 索引
	if err := rag.DeleteIndex(ctx, fileRecord.ID); err != nil {
		log.Printf("Failed to delete Redis index: %v", err)
		// 继续执行
	}

	// 删除数据库记录
	if err := file.DeleteByRefID(fileRecord.ID); err != nil {
		log.Printf("Failed to delete file record: %v", err)
		return err
	}

	log.Printf("File deleted successfully: %s", fileRecord.ObjectName)
	return nil
}

// GetFileURL 获取文件访问 URL
func GetFileURL(fileID string, userRefID uint) (string, error) {
	fileRecord, err := file.GetByFileIDAndUserRefID(fileID, userRefID)
	if err != nil {
		return "", err
	}
	if fileRecord == nil {
		return "", fmt.Errorf("文件不存在或无权限访问")
	}

	ctx := context.Background()
	url, err := minio.GetFileURL(ctx, fileRecord.ObjectName)
	if err != nil {
		return "", err
	}

	return url, nil
}

// DownloadFileContent 下载文件内容
func DownloadFileContent(fileID string, userRefID uint) (io.ReadCloser, *model.File, error) {
	fileRecord, err := file.GetByFileIDAndUserRefID(fileID, userRefID)
	if err != nil {
		return nil, nil, err
	}
	if fileRecord == nil {
		return nil, nil, fmt.Errorf("文件不存在或无权限访问")
	}

	ctx := context.Background()
	obj, err := minio.DownloadFile(ctx, fileRecord.ObjectName)
	if err != nil {
		return nil, nil, err
	}

	return obj, fileRecord, nil
}

// FormatFileSize 格式化文件大小显示
func FormatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	} else if size < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(size)/1024)
	} else if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.2f GB", float64(size)/(1024*1024*1024))
}

// FormatTime 格式化时间显示
func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
