package deepagent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/adk/filesystem"
	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"GopherAI/config"
	deepagentDAO "GopherAI/dao/deepagent"
	"GopherAI/model"
)

const (
	executionLockTTL = 6 * time.Hour
	lockPrefixExec   = "deepagent:exec:"
)

type RuntimeInfo struct {
	Status        string `json:"status"`
	ContainerName string `json:"container_name"`
	LastError     string `json:"last_error"`
	LastStartedAt string `json:"last_started_at"`
	LastUsedAt    string `json:"last_used_at"`
}

type RuntimeHandle struct {
	Runtime *model.DeepAgentRuntime
	Backend filesystem.Backend
	Shell   filesystem.Shell
}

type RuntimeManager struct {
	client      *dockerclient.Client
	lockManager *lockManager
}

var (
	runtimeManager     *RuntimeManager
	runtimeManagerOnce sync.Once
	runtimeManagerErr  error
)

func Init(ctx context.Context) error {
	if !FeatureEnabled() {
		return nil
	}
	_, err := getRuntimeManager(ctx)
	return err
}

func StartReaper(ctx context.Context) {
	if !FeatureEnabled() {
		return
	}
	manager, err := getRuntimeManager(ctx)
	if err != nil {
		return
	}

	interval := time.Duration(config.GetConfig().DeepAgentConfig.ReaperIntervalSecs) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				manager.ReapIdleRuntimes(context.Background())
			}
		}
	}()
}

func GetRuntimeManager() (*RuntimeManager, error) {
	return getRuntimeManager(context.Background())
}

func getRuntimeManager(ctx context.Context) (*RuntimeManager, error) {
	runtimeManagerOnce.Do(func() {
		if !FeatureEnabled() {
			runtimeManagerErr = fmt.Errorf("deepagent disabled")
			return
		}

		workspaceRoot, err := workspaceRootPath()
		if err != nil {
			runtimeManagerErr = err
			return
		}
		if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
			runtimeManagerErr = err
			return
		}

		opts := []dockerclient.Opt{
			dockerclient.FromEnv,
			dockerclient.WithAPIVersionNegotiation(),
		}
		if host := strings.TrimSpace(config.GetConfig().DeepAgentConfig.DockerHost); host != "" {
			opts = append(opts, dockerclient.WithHost(host))
		}

		cli, err := dockerclient.NewClientWithOpts(opts...)
		if err != nil {
			runtimeManagerErr = err
			return
		}
		if _, err := cli.Ping(ctx); err != nil {
			runtimeManagerErr = err
			return
		}

		runtimeManager = &RuntimeManager{
			client:      cli,
			lockManager: newLockManager(),
		}
	})

	if runtimeManagerErr != nil {
		return nil, runtimeManagerErr
	}
	return runtimeManager, nil
}

func (m *RuntimeManager) AcquireExecution(userRefID uint) (func(), error) {
	return m.lockManager.Acquire(context.Background(), lockPrefixExec+fmt.Sprint(userRefID), executionLockTTL)
}

func (m *RuntimeManager) acquireAdmin(userRefID uint) (func(), error) {
	return m.lockManager.Acquire(context.Background(), lockPrefixExec+fmt.Sprint(userRefID), executionLockTTL)
}

func (m *RuntimeManager) GetRuntimeInfo(ctx context.Context, userRefID uint, userUUID string) (*RuntimeInfo, error) {
	runtime, err := m.loadOrCreateRuntime(ctx, userRefID, userUUID)
	if err != nil {
		return nil, err
	}
	if err := m.refreshRuntimeStatus(ctx, runtime); err != nil {
		return nil, err
	}
	return buildRuntimeInfo(runtime), nil
}

func (m *RuntimeManager) BuildHandle(ctx context.Context, userRefID uint, userUUID string) (*RuntimeHandle, error) {
	runtime, err := m.EnsureRuntime(ctx, userRefID, userUUID)
	if err != nil {
		return nil, err
	}
	return &RuntimeHandle{
		Runtime: runtime,
		Backend: NewRootBackend(runtime.WorkspacePath),
		Shell: &containerShell{
			manager:   m,
			userRefID: userRefID,
			userUUID:  userUUID,
		},
	}, nil
}

func (m *RuntimeManager) EnsureRuntime(ctx context.Context, userRefID uint, userUUID string) (*model.DeepAgentRuntime, error) {
	runtime, err := m.loadOrCreateRuntime(ctx, userRefID, userUUID)
	if err != nil {
		return nil, err
	}

	if err := ensureWorkspace(runtime.WorkspacePath); err != nil {
		runtime.Status = model.DeepAgentRuntimeStatusError
		runtime.LastError = err.Error()
		_ = deepagentDAO.Save(runtime)
		return nil, err
	}

	if err := m.ensureContainer(ctx, runtime); err != nil {
		runtime.Status = model.DeepAgentRuntimeStatusError
		runtime.LastError = err.Error()
		_ = deepagentDAO.Save(runtime)
		return nil, err
	}

	now := time.Now().UTC()
	runtime.Status = model.DeepAgentRuntimeStatusRunning
	runtime.LastError = ""
	runtime.LastUsedAt = &now
	if runtime.LastStartedAt == nil {
		runtime.LastStartedAt = &now
	}
	if err := deepagentDAO.Save(runtime); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (m *RuntimeManager) RestartRuntime(ctx context.Context, userRefID uint, userUUID string) (*RuntimeInfo, error) {
	release, err := m.acquireAdmin(userRefID)
	if err != nil {
		return nil, err
	}
	defer release()

	runtime, err := m.loadOrCreateRuntime(ctx, userRefID, userUUID)
	if err != nil {
		return nil, err
	}
	if err := ensureWorkspace(runtime.WorkspacePath); err != nil {
		return nil, err
	}
	if err := m.removeContainer(ctx, runtime); err != nil {
		return nil, err
	}
	if err := m.ensureContainer(ctx, runtime); err != nil {
		runtime.Status = model.DeepAgentRuntimeStatusError
		runtime.LastError = err.Error()
		_ = deepagentDAO.Save(runtime)
		return nil, err
	}

	now := time.Now().UTC()
	runtime.Status = model.DeepAgentRuntimeStatusRunning
	runtime.LastError = ""
	runtime.LastStartedAt = &now
	runtime.LastUsedAt = &now
	if err := deepagentDAO.Save(runtime); err != nil {
		return nil, err
	}
	return buildRuntimeInfo(runtime), nil
}

func (m *RuntimeManager) RebuildRuntime(ctx context.Context, userRefID uint, userUUID string) (*RuntimeInfo, error) {
	release, err := m.acquireAdmin(userRefID)
	if err != nil {
		return nil, err
	}
	defer release()

	runtime, err := m.loadOrCreateRuntime(ctx, userRefID, userUUID)
	if err != nil {
		return nil, err
	}
	runtime.Status = model.DeepAgentRuntimeStatusRebuilding
	runtime.LastError = ""
	if err := deepagentDAO.Save(runtime); err != nil {
		return nil, err
	}

	if err := m.removeContainer(ctx, runtime); err != nil {
		return nil, err
	}
	if err := rebuildWorkspace(runtime.WorkspacePath); err != nil {
		runtime.Status = model.DeepAgentRuntimeStatusError
		runtime.LastError = err.Error()
		_ = deepagentDAO.Save(runtime)
		return nil, err
	}
	if err := m.ensureContainer(ctx, runtime); err != nil {
		runtime.Status = model.DeepAgentRuntimeStatusError
		runtime.LastError = err.Error()
		_ = deepagentDAO.Save(runtime)
		return nil, err
	}

	now := time.Now().UTC()
	runtime.Status = model.DeepAgentRuntimeStatusRunning
	runtime.LastError = ""
	runtime.LastStartedAt = &now
	runtime.LastUsedAt = &now
	if err := deepagentDAO.Save(runtime); err != nil {
		return nil, err
	}
	return buildRuntimeInfo(runtime), nil
}

func (m *RuntimeManager) ExecInContainer(ctx context.Context, userRefID uint, userUUID, command string) (*filesystem.ExecuteResponse, error) {
	runtime, err := m.EnsureRuntime(ctx, userRefID, userUUID)
	if err != nil {
		return nil, err
	}

	execResp, err := m.client.ContainerExecCreate(ctx, runtime.ContainerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"bash", "-lc", command},
		WorkingDir:   config.GetConfig().DeepAgentConfig.ContainerWorkdir,
	})
	if err != nil {
		return nil, err
	}

	attachResp, err := m.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, err
	}
	defer attachResp.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader); err != nil && err != io.EOF {
		return nil, err
	}

	inspectResp, err := m.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, err
	}
	exitCode := inspectResp.ExitCode
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		output += stderr.String()
	}

	now := time.Now().UTC()
	runtime.LastUsedAt = &now
	_ = deepagentDAO.Save(runtime)

	return &filesystem.ExecuteResponse{
		Output:    output,
		ExitCode:  &exitCode,
		Truncated: false,
	}, nil
}

func (m *RuntimeManager) ReapIdleRuntimes(ctx context.Context) {
	ttl := time.Duration(config.GetConfig().DeepAgentConfig.IdleTTLMinutes) * time.Minute
	if ttl <= 0 {
		return
	}

	deadline := time.Now().UTC().Add(-ttl)
	runtimes, err := deepagentDAO.ListRunningIdleBefore(deadline)
	if err != nil {
		return
	}

	for i := range runtimes {
		runtime := runtimes[i]
		release, lockErr := m.acquireAdmin(runtime.UserRefID)
		if lockErr != nil {
			continue
		}
		_ = m.removeContainer(ctx, &runtime)
		runtime.Status = model.DeepAgentRuntimeStatusStopped
		runtime.LastError = ""
		_ = deepagentDAO.Save(&runtime)
		release()
	}
}

func (m *RuntimeManager) loadOrCreateRuntime(ctx context.Context, userRefID uint, userUUID string) (*model.DeepAgentRuntime, error) {
	workspacePath, err := workspacePathForUser(userUUID)
	if err != nil {
		return nil, err
	}

	runtime, err := deepagentDAO.GetByUserRefID(userRefID)
	if err == nil {
		changed := false
		if runtime.ContainerName == "" {
			runtime.ContainerName = containerNameForUser(userRefID)
			changed = true
		}
		if runtime.WorkspacePath != workspacePath {
			runtime.WorkspacePath = workspacePath
			changed = true
			if err := m.removeContainer(ctx, runtime); err != nil {
				return nil, err
			}
		}
		if changed {
			if err := deepagentDAO.Save(runtime); err != nil {
				return nil, err
			}
		}
		return runtime, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	runtime = &model.DeepAgentRuntime{
		DeepAgentRuntimeID: uuid.NewString(),
		UserRefID:          userRefID,
		Status:             model.DeepAgentRuntimeStatusStopped,
		ContainerName:      containerNameForUser(userRefID),
		WorkspacePath:      workspacePath,
	}
	if err := deepagentDAO.Create(runtime); err != nil {
		return nil, err
	}
	return runtime, nil
}

func (m *RuntimeManager) refreshRuntimeStatus(ctx context.Context, runtime *model.DeepAgentRuntime) error {
	if runtime == nil {
		return nil
	}
	if runtime.ContainerID == "" {
		runtime.Status = model.DeepAgentRuntimeStatusStopped
		return deepagentDAO.Save(runtime)
	}

	inspectResp, err := m.client.ContainerInspect(ctx, runtime.ContainerID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			runtime.Status = model.DeepAgentRuntimeStatusStopped
			runtime.ContainerID = ""
			return deepagentDAO.Save(runtime)
		}
		return err
	}

	if inspectResp.State != nil && inspectResp.State.Running {
		runtime.Status = model.DeepAgentRuntimeStatusRunning
	} else {
		runtime.Status = model.DeepAgentRuntimeStatusStopped
	}
	return deepagentDAO.Save(runtime)
}

func (m *RuntimeManager) ensureContainer(ctx context.Context, runtime *model.DeepAgentRuntime) error {
	runtime.Status = model.DeepAgentRuntimeStatusStarting
	runtime.LastError = ""
	if err := deepagentDAO.Save(runtime); err != nil {
		return err
	}

	if runtime.ContainerID != "" {
		inspectResp, err := m.client.ContainerInspect(ctx, runtime.ContainerID)
		if err == nil {
			if inspectResp.State != nil && inspectResp.State.Running {
				return nil
			}
			if startErr := m.client.ContainerStart(ctx, runtime.ContainerID, container.StartOptions{}); startErr == nil {
				now := time.Now().UTC()
				runtime.LastStartedAt = &now
				return nil
			}
		}
	}

	inspectByName, err := m.client.ContainerInspect(ctx, runtime.ContainerName)
	if err == nil {
		runtime.ContainerID = inspectByName.ID
		if inspectByName.State != nil && inspectByName.State.Running {
			return nil
		}
		if err := m.client.ContainerStart(ctx, inspectByName.ID, container.StartOptions{}); err == nil {
			now := time.Now().UTC()
			runtime.LastStartedAt = &now
			return nil
		}
	}

	if err := m.removeContainer(ctx, runtime); err != nil {
		return err
	}

	resp, err := m.client.ContainerCreate(ctx, &container.Config{
		Image:      config.GetConfig().DeepAgentConfig.Image,
		Cmd:        []string{"sleep", "infinity"},
		WorkingDir: config.GetConfig().DeepAgentConfig.ContainerWorkdir,
	}, &container.HostConfig{
		Binds: []string{fmt.Sprintf("%s:%s", runtime.WorkspacePath, config.GetConfig().DeepAgentConfig.ContainerWorkdir)},
	}, nil, nil, runtime.ContainerName)
	if err != nil {
		return err
	}
	if err := m.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	now := time.Now().UTC()
	runtime.ContainerID = resp.ID
	runtime.LastStartedAt = &now
	return nil
}

func (m *RuntimeManager) removeContainer(ctx context.Context, runtime *model.DeepAgentRuntime) error {
	if runtime == nil {
		return nil
	}

	targets := []string{}
	if runtime.ContainerID != "" {
		targets = append(targets, runtime.ContainerID)
	}
	if runtime.ContainerName != "" && (len(targets) == 0 || targets[0] != runtime.ContainerName) {
		targets = append(targets, runtime.ContainerName)
	}

	for _, target := range targets {
		if target == "" {
			continue
		}
		if err := m.client.ContainerRemove(ctx, target, container.RemoveOptions{Force: true}); err != nil && !errdefs.IsNotFound(err) {
			return err
		}
	}

	runtime.ContainerID = ""
	runtime.Status = model.DeepAgentRuntimeStatusStopped
	return deepagentDAO.Save(runtime)
}

type containerShell struct {
	manager   *RuntimeManager
	userRefID uint
	userUUID  string
}

func (s *containerShell) Execute(ctx context.Context, input *filesystem.ExecuteRequest) (*filesystem.ExecuteResponse, error) {
	if input == nil {
		return nil, fmt.Errorf("execute request is nil")
	}
	if input.RunInBackendGround {
		return nil, fmt.Errorf("background execution is not supported")
	}
	return s.manager.ExecInContainer(ctx, s.userRefID, s.userUUID, input.Command)
}

func buildRuntimeInfo(runtime *model.DeepAgentRuntime) *RuntimeInfo {
	if runtime == nil {
		return &RuntimeInfo{
			Status: model.DeepAgentRuntimeStatusStopped,
		}
	}

	return &RuntimeInfo{
		Status:        runtime.Status,
		ContainerName: runtime.ContainerName,
		LastError:     runtime.LastError,
		LastStartedAt: formatTime(runtime.LastStartedAt),
		LastUsedAt:    formatTime(runtime.LastUsedAt),
	}
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
