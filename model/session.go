package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// Session 会话模型
type Session struct {
	ID        string          `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserName  string          `gorm:"index;not null" json:"username"`
	Title     string          `gorm:"type:varchar(100)" json:"title"`
	Tools     json.RawMessage `gorm:"type:jsonb" json:"tools,omitempty"` // 工具配置
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt gorm.DeletedAt  `gorm:"index" json:"-"`
}

// SessionInfo 会话信息（API 返回）
type SessionInfo struct {
	SessionID string `json:"sessionId"`
	Title     string `json:"title"`
}

// GetTools 获取工具列表
func (s *Session) GetTools() []string {
	if s.Tools == nil {
		return nil
	}
	var tools []string
	json.Unmarshal(s.Tools, &tools)
	return tools
}

// SetTools 设置工具列表
func (s *Session) SetTools(tools []string) error {
	data, err := json.Marshal(tools)
	if err != nil {
		return err
	}
	s.Tools = data
	return nil
}
