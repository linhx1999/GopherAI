package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	eino_model "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"

	"GopherAI/common/agent/tools"
	"GopherAI/common/llm"
	"GopherAI/dao/file"
)

// AgentManager Agent 管理器
type AgentManager struct {
	mu     sync.RWMutex
	agents map[string]*react.Agent // cacheKey -> Agent
}

type AgentSessionConfig struct {
	EnabledToolNames []string
	Model            eino_model.ToolCallingChatModel
	ModelName        string
	ToolInstances    []tool.BaseTool
	ToolSignature    string
}

var (
	manager     *AgentManager
	managerOnce sync.Once
)

// GetAgentManager 获取全局 Agent 管理器
func GetAgentManager() *AgentManager {
	managerOnce.Do(func() {
		manager = &AgentManager{
			agents: make(map[string]*react.Agent),
		}
	})
	return manager
}

func buildAgentCacheKey(sessionID string, modelName string, toolSignature string) string {
	return fmt.Sprintf("%s::%s::%s", sessionID, modelName, toolSignature)
}

func buildToolSignature(enabledToolNames []string) string {
	normalizedNames := tools.NormalizeToolNames(enabledToolNames)
	if len(normalizedNames) == 0 {
		return "no-tools"
	}
	return strings.Join(normalizedNames, ",")
}

func (m *AgentManager) resolveAgentSessionConfig(ctx context.Context, userName string, enabledToolNames []string, thinkingMode bool) (*AgentSessionConfig, error) {
	client := llm.GetLLMClient()
	if client == nil {
		return nil, fmt.Errorf("llm client not initialized")
	}

	model, modelName, err := client.GetModelForMode(ctx, thinkingMode)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, fmt.Errorf("llm model not available")
	}

	fileIDs, err := file.GetIndexedFileIDsByUserName(userName)
	if err != nil {
		log.Printf("Warning: failed to get indexed file ids: %v", err)
		fileIDs = []uint{}
	}

	registry := tools.GetToolRegistry()
	normalizedToolNames := tools.NormalizeToolNames(enabledToolNames)
	toolInstances, err := registry.ResolveTools(ctx, normalizedToolNames, fileIDs)
	if err != nil {
		return nil, fmt.Errorf("resolve tools failed: %w", err)
	}

	return &AgentSessionConfig{
		EnabledToolNames: normalizedToolNames,
		Model:            model,
		ModelName:        modelName,
		ToolInstances:    toolInstances,
		ToolSignature:    buildToolSignature(normalizedToolNames),
	}, nil
}

func (m *AgentManager) buildAgent(ctx context.Context, config *AgentSessionConfig) (*react.Agent, error) {
	if config == nil || config.Model == nil {
		return nil, fmt.Errorf("agent config not available")
	}

	log.Printf("Creating agent with model=%s, tools=%v", config.ModelName, config.EnabledToolNames)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: config.Model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: config.ToolInstances,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create react agent failed: %w", err)
	}

	return agent, nil
}

// GetOrCreateAgentForChat 获取或创建指定模型和工具配置的 Agent。
func (m *AgentManager) GetOrCreateAgentForChat(ctx context.Context, sessionID string, userName string, enabledToolNames []string, thinkingMode bool) (*react.Agent, error) {
	config, err := m.resolveAgentSessionConfig(ctx, userName, enabledToolNames, thinkingMode)
	if err != nil {
		return nil, err
	}
	cacheKey := buildAgentCacheKey(sessionID, config.ModelName, config.ToolSignature)

	m.mu.RLock()
	if agent, ok := m.agents[cacheKey]; ok {
		m.mu.RUnlock()
		return agent, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if agent, ok := m.agents[cacheKey]; ok {
		return agent, nil
	}

	agent, err := m.buildAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	m.agents[cacheKey] = agent
	return agent, nil
}

// ClearAgent 清除指定会话的 Agent 缓存
func (m *AgentManager) ClearAgent(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, sessionID)
	prefix := sessionID + "::"
	for key := range m.agents {
		if strings.HasPrefix(key, prefix) {
			delete(m.agents, key)
		}
	}
}
