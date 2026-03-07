package model

import (
	"encoding/json"
	"time"
)

// ToolCall 工具调用记录
type ToolCall struct {
	ToolID    string          `json:"tool_id"`
	Function  string          `json:"function"`
	Arguments json.RawMessage `json:"arguments"` // 原始 JSON 参数
	Result    string          `json:"result,omitempty"`
}

// ToolCalls 工具调用列表
type ToolCalls []ToolCall

// Message 消息模型
type Message struct {
	ID        uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	SessionID string          `gorm:"index;not null;type:varchar(36)" json:"session_id"`
	UserName  string          `gorm:"type:varchar(20)" json:"username"`
	Index     int             `gorm:"not null;index:idx_session_index,priority:2" json:"index"` // 线性索引
	Role      string          `gorm:"type:varchar(20);not null;default:'user'" json:"role"`     // user/assistant
	Content   string          `gorm:"type:text" json:"content"`
	ToolCalls json.RawMessage `gorm:"type:jsonb" json:"tool_calls,omitempty"` // 工具调用记录
	CreatedAt time.Time       `json:"created_at"`
}

// GetToolCalls 解析工具调用
func (m *Message) GetToolCalls() ToolCalls {
	if m.ToolCalls == nil {
		return nil
	}
	var calls ToolCalls
	json.Unmarshal(m.ToolCalls, &calls)
	return calls
}

// SetToolCalls 设置工具调用
func (m *Message) SetToolCalls(calls ToolCalls) error {
	data, err := json.Marshal(calls)
	if err != nil {
		return err
	}
	m.ToolCalls = data
	return nil
}
