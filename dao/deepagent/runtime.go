package deepagent

import (
	"time"

	"GopherAI/common/postgres"
	"GopherAI/model"
)

func GetByUserRefID(userRefID uint) (*model.DeepAgentRuntime, error) {
	var runtime model.DeepAgentRuntime
	err := postgres.DB.Where("user_ref_id = ?", userRefID).First(&runtime).Error
	if err != nil {
		return nil, err
	}
	return &runtime, nil
}

func Save(runtime *model.DeepAgentRuntime) error {
	return postgres.DB.Save(runtime).Error
}

func Create(runtime *model.DeepAgentRuntime) error {
	return postgres.DB.Create(runtime).Error
}

func ListRunningIdleBefore(deadline time.Time) ([]model.DeepAgentRuntime, error) {
	var runtimes []model.DeepAgentRuntime
	err := postgres.DB.
		Where("status = ? AND last_used_at IS NOT NULL AND last_used_at < ?", model.DeepAgentRuntimeStatusRunning, deadline).
		Find(&runtimes).Error
	return runtimes, err
}
