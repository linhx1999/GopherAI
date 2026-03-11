package model

import (
	"gorm.io/gorm"
)

// Session 会话模型
type Session struct {
	gorm.Model
	SessionID string `gorm:"column:session_id;type:varchar(36);uniqueIndex;not null" json:"session_id"`
	UserName  string `gorm:"index;not null" json:"username"`
	Title     string `gorm:"type:varchar(100)" json:"title"`
}

// SessionInfo 会话信息（API 返回）
type SessionInfo struct {
	SessionID string `json:"sessionId"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
}
