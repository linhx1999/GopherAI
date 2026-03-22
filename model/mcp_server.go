package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const (
	MCPTransportSSE  = "sse"
	MCPTransportHTTP = "http"

	MCPTestStatusUntested = "untested"
	MCPTestStatusSuccess  = "success"
	MCPTestStatusFailed   = "failed"
)

type MCPServer struct {
	gorm.Model
	MCPServerID       string          `gorm:"column:mcp_server_id;type:varchar(36);uniqueIndex;not null" json:"mcp_server_id"`
	UserRefID         uint            `gorm:"column:user_ref_id;index;not null" json:"-"`
	Name              string          `gorm:"type:varchar(100);not null" json:"name"`
	TransportType     string          `gorm:"column:transport_type;type:varchar(20);not null;default:'sse'" json:"transport_type"`
	Endpoint          string          `gorm:"type:text;not null" json:"endpoint"`
	HeadersCiphertext string          `gorm:"column:headers_ciphertext;type:text" json:"-"`
	ToolSnapshot      json.RawMessage `gorm:"column:tool_snapshot;type:jsonb" json:"tool_snapshot,omitempty"`
	LastTestStatus    string          `gorm:"column:last_test_status;type:varchar(20);not null;default:'untested'" json:"last_test_status"`
	LastTestMessage   string          `gorm:"column:last_test_message;type:varchar(500)" json:"last_test_message"`
	LastTestedAt      *time.Time      `gorm:"column:last_tested_at" json:"last_tested_at"`
}
