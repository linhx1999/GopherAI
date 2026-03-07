package model

import (
	"github.com/pgvector/pgvector-go"
)

// DocumentChunk 文档分块模型，用于 RAG 向量检索
type DocumentChunk struct {
	ID       uint           `gorm:"primaryKey" json:"id"`
	FileID   uint           `gorm:"not null;index" json:"file_id"`                             // 关联的文件ID
	Content  string         `gorm:"type:text;not null" json:"content"`                         // 文档内容
	Embedding pgvector.Vector `gorm:"type:vector(1024)" json:"-"`                               // 向量嵌入（维度根据 embedding 模型调整）
	Metadata string         `gorm:"type:text" json:"metadata"`                                 // 元数据（JSON 格式）
	CreatedAt int64         `gorm:"autoCreateTime" json:"created_at"`
}

// TableName 指定表名
func (DocumentChunk) TableName() string {
	return "document_chunks"
}

// DocumentChunkWithDistance 带相似度距离的结果
type DocumentChunkWithDistance struct {
	DocumentChunk
	Distance float64 `json:"distance"` // 余弦距离
}
