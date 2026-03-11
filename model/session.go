package model

import (
	"gorm.io/gorm"
)

// Session 会话模型
type Session struct {
	gorm.Model
	SessionID string `gorm:"column:session_id;type:varchar(36);uniqueIndex;not null" json:"session_id"`
	UserRefID uint   `gorm:"column:user_ref_id;index;not null" json:"-"`
	Title     string `gorm:"type:varchar(100)" json:"title"`
}

// SessionInfo 会话信息（API 返回）
type SessionInfo struct {
	SessionID string `json:"sessionId"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
}
