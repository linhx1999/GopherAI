package rag

import (
	"GopherAI/common/minio"
	"GopherAI/common/rag"
	"GopherAI/config"
	"GopherAI/dao/file"
	fileModel "GopherAI/model"
	"context"
	"fmt"
	"io"
	"log"
)

// RAGService 文件索引服务
type RAGService struct{}

// IndexFile 索引指定文件
func (s *RAGService) IndexFile(fileID uint) error {
	ctx := context.Background()

	// 1. 获取文件记录
	fileRecord, err := file.GetByID(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	// 2. 更新状态为 indexing
	if err := file.UpdateIndexStatus(fileID, fileModel.IndexStatusIndexing, "开始索引"); err != nil {
		log.Printf("Failed to update index status to indexing: %v", err)
	}

	// 3. 从 MinIO 下载文件内容
	obj, err := minio.DownloadFile(ctx, fileRecord.ObjectName)
	if err != nil {
		errMsg := fmt.Sprintf("下载文件失败: %v", err)
		file.UpdateIndexStatus(fileID, fileModel.IndexStatusFailed, errMsg)
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer obj.Close()

	// 4. 读取文件内容
	content, err := io.ReadAll(obj)
	if err != nil {
		errMsg := fmt.Sprintf("读取文件内容失败: %v", err)
		file.UpdateIndexStatus(fileID, fileModel.IndexStatusFailed, errMsg)
		return fmt.Errorf("failed to read file content: %w", err)
	}

	// 5. 创建 RAG 索引器
	indexer, err := rag.NewRAGIndexer(
		ctx,
		fileID,
		content,
		fileRecord.FileType,
		config.GetConfig().RagModelConfig.RagEmbeddingModel,
	)
	if err != nil {
		errMsg := fmt.Sprintf("创建索引器失败: %v", err)
		file.UpdateIndexStatus(fileID, fileModel.IndexStatusFailed, errMsg)
		return fmt.Errorf("failed to create indexer: %w", err)
	}

	// 6. 执行索引（包含文档切分和向量化）
	if err := indexer.IndexFileContent(ctx); err != nil {
		errMsg := fmt.Sprintf("索引失败: %v", err)
		file.UpdateIndexStatus(fileID, fileModel.IndexStatusFailed, errMsg)
		return fmt.Errorf("failed to index file content: %w", err)
	}

	// 7. 更新状态为 indexed
	if err := file.UpdateIndexStatus(fileID, fileModel.IndexStatusIndexed, "索引完成"); err != nil {
		log.Printf("Failed to update index status to indexed: %v", err)
	}

	log.Printf("File indexed successfully: ID=%d, Name=%s", fileID, fileRecord.FileName)
	return nil
}

// DeleteIndex 删除文件索引
func (s *RAGService) DeleteIndex(fileID uint) error {
	ctx := context.Background()

	// 删除 Redis 索引
	if err := rag.DeleteIndex(ctx, fileID); err != nil {
		log.Printf("Failed to delete redis index: %v", err)
		// 继续执行，更新数据库状态
	}

	// 更新状态为 pending
	if err := file.UpdateIndexStatus(fileID, fileModel.IndexStatusPending, "索引已删除"); err != nil {
		return fmt.Errorf("failed to update index status: %w", err)
	}

	return nil
}