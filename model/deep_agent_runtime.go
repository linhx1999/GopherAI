package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	DeepAgentRuntimeStatusStopped    = "stopped"
	DeepAgentRuntimeStatusStarting   = "starting"
	DeepAgentRuntimeStatusRunning    = "running"
	DeepAgentRuntimeStatusError      = "error"
	DeepAgentRuntimeStatusRebuilding = "rebuilding"
)

type DeepAgentRuntime struct {
	gorm.Model
	DeepAgentRuntimeID string     `gorm:"column:deep_agent_runtime_id;type:varchar(36);uniqueIndex;not null" json:"deep_agent_runtime_id"`
	UserRefID          uint       `gorm:"column:user_ref_id;uniqueIndex;not null" json:"-"`
	Status             string     `gorm:"column:status;type:varchar(20);not null;default:'stopped'" json:"status"`
	ContainerName      string     `gorm:"column:container_name;type:varchar(200);not null" json:"container_name"`
	ContainerID        string     `gorm:"column:container_id;type:varchar(120)" json:"container_id"`
	WorkspacePath      string     `gorm:"column:workspace_path;type:text;not null" json:"workspace_path"`
	LastError          string     `gorm:"column:last_error;type:text" json:"last_error"`
	LastStartedAt      *time.Time `gorm:"column:last_started_at" json:"last_started_at"`
	LastUsedAt         *time.Time `gorm:"column:last_used_at" json:"last_used_at"`
}
