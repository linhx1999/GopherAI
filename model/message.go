package model

import (
	"encoding/json"
	"time"

	"github.com/cloudwego/eino/schema"
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
	Payload   json.RawMessage `gorm:"type:jsonb" json:"payload,omitempty"`    // 完整 schema.Message
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

// GetSchemaMessage 获取完整 schema.Message；当 payload 缺失时回退到旧字段。
func (m *Message) GetSchemaMessage() *schema.Message {
	if len(m.Payload) > 0 {
		var msg schema.Message
		if err := json.Unmarshal(m.Payload, &msg); err == nil {
			return &msg
		}
	}

	msg := &schema.Message{
		Role:    schema.RoleType(m.Role),
		Content: m.Content,
	}
	if len(m.ToolCalls) == 0 {
		return msg
	}

	toolCalls := m.GetToolCalls()
	if len(toolCalls) == 0 {
		return msg
	}

	msg.ToolCalls = make([]schema.ToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		msg.ToolCalls = append(msg.ToolCalls, schema.ToolCall{
			ID:   tc.ToolID,
			Type: "function",
			Function: schema.FunctionCall{
				Name:      tc.Function,
				Arguments: string(tc.Arguments),
			},
		})
	}
	return msg
}

// SetSchemaMessage 保存完整 schema.Message，并同步常用冗余字段。
func (m *Message) SetSchemaMessage(msg *schema.Message) error {
	if msg == nil {
		m.Payload = nil
		m.ToolCalls = nil
		m.Role = ""
		m.Content = ""
		return nil
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	m.Payload = data
	m.Role = string(msg.Role)
	m.Content = msg.Content

	if len(msg.ToolCalls) == 0 {
		m.ToolCalls = nil
		return nil
	}

	calls := make(ToolCalls, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		calls = append(calls, ToolCall{
			ToolID:    tc.ID,
			Function:  tc.Function.Name,
			Arguments: json.RawMessage(tc.Function.Arguments),
		})
	}
	return m.SetToolCalls(calls)
}
