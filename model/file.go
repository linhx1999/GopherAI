package model

import (
	"time"

	"gorm.io/gorm"
)

// 索引状态常量
const (
	IndexStatusPending  = "pending"  // 待索引
	IndexStatusIndexing = "indexing" // 索引中
	IndexStatusIndexed  = "indexed"  // 已索引
	IndexStatusFailed   = "failed"   // 索引失败
)

// File 文件元数据模型
type File struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	UserName     string         `gorm:"type:varchar(50);index" json:"user_name"`             // 用户名
	FileName     string         `gorm:"type:varchar(255)" json:"file_name"`                  // 原始文件名
	ObjectName   string         `gorm:"type:varchar(255);uniqueIndex" json:"object_name"`     // MinIO 对象名（唯一）
	FileSize     int64          `json:"file_size"`                                          // 文件大小（字节）
	FileType     string         `gorm:"type:varchar(50)" json:"file_type"`                   // 文件类型（扩展名）
	Bucket       string         `gorm:"type:varchar(100)" json:"bucket"`                     // 存储桶名称
	IndexStatus  string         `gorm:"type:varchar(20);default:'pending'" json:"index_status"` // 索引状态：pending/indexing/indexed/failed
	IndexMessage string         `gorm:"type:varchar(255)" json:"index_message"`              // 索引状态消息（错误信息等）
	CreatedAt    time.Time      `json:"created_at"`                                        // 上传时间
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`                                    // 软删除
}
