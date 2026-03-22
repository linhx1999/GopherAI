package deepagent

import (
	"context"
	"log"
	"strings"

	"GopherAI/common/code"
	commondeep "GopherAI/common/deepagent"
)

type RuntimeStatus struct {
	Status        string `json:"status"`
	ContainerName string `json:"container_name"`
	LastError     string `json:"last_error"`
	LastStartedAt string `json:"last_started_at"`
	LastUsedAt    string `json:"last_used_at"`
}

func FeatureEnabled() bool {
	return commondeep.FeatureEnabled()
}

func GetRuntimeStatus(ctx context.Context, userRefID uint, userUUID string) (*RuntimeStatus, code.Code) {
	if !FeatureEnabled() {
		return nil, code.CodeDeepAgentFeatureDisabled
	}

	manager, err := commondeep.GetRuntimeManager()
	if err != nil {
		log.Printf("GetRuntimeStatus manager error: %v", err)
		return nil, code.CodeDeepAgentContainerFailed
	}

	info, err := manager.GetRuntimeInfo(ctx, userRefID, userUUID)
	if err != nil {
		log.Printf("GetRuntimeStatus error: %v", err)
		return nil, code.CodeDeepAgentContainerFailed
	}
	return convertRuntimeInfo(info), code.CodeSuccess
}

func RestartRuntime(ctx context.Context, userRefID uint, userUUID string) (*RuntimeStatus, code.Code) {
	return mutateRuntime(ctx, userRefID, userUUID, func(manager *commondeep.RuntimeManager) (*commondeep.RuntimeInfo, error) {
		return manager.RestartRuntime(ctx, userRefID, userUUID)
	})
}

func RebuildRuntime(ctx context.Context, userRefID uint, userUUID string) (*RuntimeStatus, code.Code) {
	return mutateRuntime(ctx, userRefID, userUUID, func(manager *commondeep.RuntimeManager) (*commondeep.RuntimeInfo, error) {
		return manager.RebuildRuntime(ctx, userRefID, userUUID)
	})
}

func mutateRuntime(ctx context.Context, userRefID uint, userUUID string, action func(manager *commondeep.RuntimeManager) (*commondeep.RuntimeInfo, error)) (*RuntimeStatus, code.Code) {
	if !FeatureEnabled() {
		return nil, code.CodeDeepAgentFeatureDisabled
	}

	manager, err := commondeep.GetRuntimeManager()
	if err != nil {
		log.Printf("mutateRuntime manager error: %v", err)
		return nil, code.CodeDeepAgentContainerFailed
	}

	info, err := action(manager)
	if err != nil {
		log.Printf("mutateRuntime error: %v", err)
		if strings.Contains(strings.ToLower(err.Error()), "lock busy") {
			return nil, code.CodeDeepAgentRuntimeBusy
		}
		return nil, code.CodeDeepAgentContainerFailed
	}
	return convertRuntimeInfo(info), code.CodeSuccess
}

func convertRuntimeInfo(info *commondeep.RuntimeInfo) *RuntimeStatus {
	if info == nil {
		return &RuntimeStatus{Status: "stopped"}
	}
	return &RuntimeStatus{
		Status:        info.Status,
		ContainerName: info.ContainerName,
		LastError:     info.LastError,
		LastStartedAt: info.LastStartedAt,
		LastUsedAt:    info.LastUsedAt,
	}
}
