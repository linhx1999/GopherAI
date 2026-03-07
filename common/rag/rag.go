package rag

import (
	"GopherAI/common/postgres"
	"GopherAI/config"
	"GopherAI/model"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	embeddingArk "github.com/cloudwego/eino-ext/components/embedding/ark"
	splitterMarkdown "github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	splitterRecursive "github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/schema"
	"github.com/pgvector/pgvector-go"
)

// RAGIndexer 索引器 - 负责文档切分和向量化存储
type RAGIndexer struct {
	embedding   *embeddingArk.Embedder
	splitter    document.Transformer
	fileID      uint
	fileContent []byte
	fileType    string
	dimension   int
}

// RAGQuery 查询器 - 负责向量检索和问答
type RAGQuery struct {
	embedding *embeddingArk.Embedder
	fileID    uint
	topK      int
}

// NewRAGIndexer 创建 RAG 索引器
// 用于构建知识库索引（文本解析、文本切块、向量化、存储向量）
func NewRAGIndexer(ctx context.Context, fileID uint, fileContent []byte, fileType string, embeddingModel string) (*RAGIndexer, error) {
	// 从环境变量中读取调用向量模型所需的 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")

	// 向量的维度大小（等于向量模型输出的数字个数）
	dimension := config.GetConfig().RagModelConfig.RagDimension

	// 1. 创建向量生成器（Embedding）
	embedConfig := &embeddingArk.EmbeddingConfig{
		BaseURL: config.GetConfig().RagModelConfig.RagBaseUrl,
		APIKey:  apiKey,
		Model:   embeddingModel,
	}

	embedder, err := embeddingArk.NewEmbedder(ctx, embedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// 2. 创建文档切分器
	docSplitter := createSplitter(fileType)
	if docSplitter == nil {
		return nil, fmt.Errorf("failed to create splitter for file type: %s", fileType)
	}

	return &RAGIndexer{
		embedding:   embedder,
		splitter:    docSplitter,
		fileID:      fileID,
		fileContent: fileContent,
		fileType:    fileType,
		dimension:   dimension,
	}, nil
}

// createSplitter 根据文件类型创建对应的文档切分器
func createSplitter(fileType string) document.Transformer {
	ctx := context.Background()

	switch strings.ToLower(fileType) {
	case ".md":
		// Markdown 标题切分器
		splitter, err := splitterMarkdown.NewHeaderSplitter(ctx, &splitterMarkdown.HeaderConfig{
			Headers: map[string]string{
				"#":   "h1",
				"##":  "h2",
				"###": "h3",
			},
			TrimHeaders: false,
		})
		if err != nil {
			return nil
		}
		return splitter
	case ".txt":
		// 纯文本固定长度切分器（使用 recursive splitter）
		splitter, err := splitterRecursive.NewSplitter(ctx, &splitterRecursive.Config{
			ChunkSize:   500,
			OverlapSize: 50,
			Separators:  []string{"\n\n", "\n", " ", ""},
		})
		if err != nil {
			return nil
		}
		return splitter
	default:
		// 默认使用固定长度切分器
		splitter, err := splitterRecursive.NewSplitter(ctx, &splitterRecursive.Config{
			ChunkSize:   500,
			OverlapSize: 50,
			Separators:  []string{"\n\n", "\n", " ", ""},
		})
		if err != nil {
			return nil
		}
		return splitter
	}
}

// IndexFileContent 索引文件内容（使用文档切分）
func (r *RAGIndexer) IndexFileContent(ctx context.Context) error {
	// 创建原始文档
	originalDoc := &schema.Document{
		ID:      fmt.Sprintf("file_%d", r.fileID),
		Content: string(r.fileContent),
		MetaData: map[string]any{
			"source": fmt.Sprintf("file_%d", r.fileID),
		},
	}

	// 使用切分器切分文档
	splitDocs, err := r.splitter.Transform(ctx, []*schema.Document{originalDoc})
	if err != nil {
		return fmt.Errorf("failed to split document: %w", err)
	}

	// 删除该文件的旧索引
	if err := DeleteIndex(ctx, r.fileID); err != nil {
		// 忽略删除错误（可能不存在）
	}

	// 批量处理切分后的文档
	for i, doc := range splitDocs {
		// 生成向量嵌入
		embeddings, err := r.embedding.EmbedStrings(ctx, []string{doc.Content})
		if err != nil {
			return fmt.Errorf("failed to embed document %d: %w", i, err)
		}

		if len(embeddings) == 0 {
			continue
		}

		// 将 float64 转换为 float32
		float32Embedding := make([]float32, len(embeddings[0]))
		for i, v := range embeddings[0] {
			float32Embedding[i] = float32(v)
		}
		vector := pgvector.NewVector(float32Embedding)

		// 准备元数据
		metadata := map[string]interface{}{
			"source":    fmt.Sprintf("file_%d_chunk_%d", r.fileID, i),
			"file_type": r.fileType,
		}
		metadataJSON, _ := json.Marshal(metadata)

		// 创建文档块记录
		chunk := &model.DocumentChunk{
			FileID:    r.fileID,
			Content:   doc.Content,
			Embedding: vector,
			Metadata:  string(metadataJSON),
		}

		// 存储到 PostgreSQL
		if err := postgres.DB.Create(chunk).Error; err != nil {
			return fmt.Errorf("failed to store chunk %d: %w", i, err)
		}
	}

	return nil
}

// NewRAGQuery 创建 RAG 查询器（用于向量检索和问答）
func NewRAGQuery(ctx context.Context, fileID uint) (*RAGQuery, error) {
	cfg := config.GetConfig()
	apiKey := os.Getenv("OPENAI_API_KEY")

	// 创建 embedding 模型
	embedConfig := &embeddingArk.EmbeddingConfig{
		BaseURL: cfg.RagModelConfig.RagBaseUrl,
		APIKey:  apiKey,
		Model:   cfg.RagModelConfig.RagEmbeddingModel,
	}
	embedder, err := embeddingArk.NewEmbedder(ctx, embedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	return &RAGQuery{
		embedding: embedder,
		fileID:    fileID,
		topK:      5,
	}, nil
}

// SetTopK 设置返回的文档数量
func (r *RAGQuery) SetTopK(k int) {
	r.topK = k
}

// RetrieveDocuments 检索相关文档
func (r *RAGQuery) RetrieveDocuments(ctx context.Context, query string) ([]*schema.Document, error) {
	// 生成查询向量
	embeddings, err := r.embedding.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated for query")
	}

	// 将 float64 转换为 float32
	float32Embedding := make([]float32, len(embeddings[0]))
	for i, v := range embeddings[0] {
		float32Embedding[i] = float32(v)
	}
	vector := pgvector.NewVector(float32Embedding)

	// 使用 pgvector 进行余弦相似度检索
	var results []model.DocumentChunkWithDistance
	querySQL := `
		SELECT id, file_id, content, metadata, created_at, 
			   1 - (embedding <=> ?) as distance
		FROM document_chunks
		WHERE file_id = ?
		ORDER BY embedding <=> ?
		LIMIT ?
	`
	if err := postgres.DB.Raw(querySQL, vector, r.fileID, vector).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	// 转换为 schema.Document 格式
	docs := make([]*schema.Document, 0, len(results))
	for _, result := range results {
		doc := &schema.Document{
			ID:      fmt.Sprintf("%d", result.ID),
			Content: result.Content,
			MetaData: map[string]any{
				"file_id":  result.FileID,
				"distance": result.Distance,
			},
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// RetrieveDocumentsFromMultipleFiles 从多个文件中检索相关文档
func RetrieveDocumentsFromMultipleFiles(ctx context.Context, fileIDs []uint, query string, topK int) ([]*schema.Document, error) {
	if len(fileIDs) == 0 {
		return nil, nil
	}

	cfg := config.GetConfig()
	apiKey := os.Getenv("OPENAI_API_KEY")

	// 创建 embedding 模型
	embedConfig := &embeddingArk.EmbeddingConfig{
		BaseURL: cfg.RagModelConfig.RagBaseUrl,
		APIKey:  apiKey,
		Model:   cfg.RagModelConfig.RagEmbeddingModel,
	}
	embedder, err := embeddingArk.NewEmbedder(ctx, embedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedder: %w", err)
	}

	// 生成查询向量
	embeddings, err := embedder.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated for query")
	}

	// 将 float64 转换为 float32
	float32Embedding := make([]float32, len(embeddings[0]))
	for i, v := range embeddings[0] {
		float32Embedding[i] = float32(v)
	}
	vector := pgvector.NewVector(float32Embedding)

	// 使用 pgvector 进行余弦相似度检索
	var results []model.DocumentChunkWithDistance
	querySQL := `
		SELECT id, file_id, content, metadata, created_at, 
			   1 - (embedding <=> ?) as distance
		FROM document_chunks
		WHERE file_id IN ?
		ORDER BY embedding <=> ?
		LIMIT ?
	`
	if err := postgres.DB.Raw(querySQL, vector, fileIDs, vector).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve documents: %w", err)
	}

	// 转换为 schema.Document 格式
	docs := make([]*schema.Document, 0, len(results))
	for _, result := range results {
		doc := &schema.Document{
			ID:      fmt.Sprintf("%d", result.ID),
			Content: result.Content,
			MetaData: map[string]any{
				"file_id":  result.FileID,
				"distance": result.Distance,
			},
		}
		docs = append(docs, doc)
	}

	return docs, nil
}

// BuildRAGPrompt 构建包含检索文档的提示词
func BuildRAGPrompt(query string, docs []*schema.Document) string {
	if len(docs) == 0 {
		return query
	}

	contextText := ""
	for i, doc := range docs {
		contextText += fmt.Sprintf("[文档 %d]: %s\n\n", i+1, doc.Content)
	}

	prompt := fmt.Sprintf(`基于以下参考文档回答用户的问题。如果文档中没有相关信息，请说明无法找到相关信息。

参考文档：
%s

用户问题：%s

请提供准确、完整的回答：`, contextText, query)

	return prompt
}

// DeleteIndex 删除指定文件的所有向量索引
func DeleteIndex(ctx context.Context, fileID uint) error {
	return postgres.DB.Where("file_id = ?", fileID).Delete(&model.DocumentChunk{}).Error
}

// GetChunkCount 获取指定文件的文档块数量
func GetChunkCount(fileID uint) (int64, error) {
	var count int64
	err := postgres.DB.Model(&model.DocumentChunk{}).Where("file_id = ?", fileID).Count(&count).Error
	return count, err
}
