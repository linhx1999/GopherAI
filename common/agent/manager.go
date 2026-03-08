package agent

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	eino_model "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"GopherAI/common/agent/tools"
	"GopherAI/common/llm"
	"GopherAI/dao/file"
)

// AgentManager Agent 管理器
type AgentManager struct {
	mu     sync.RWMutex
	agents map[string]*react.Agent // cacheKey -> Agent
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

func buildAgentCacheKey(sessionID string, modelName string) string {
	return fmt.Sprintf("%s::%s", sessionID, modelName)
}

// GetOrCreateAgent 获取或创建 Agent（根据用户名动态查询文件ID，使用默认工具）
func (m *AgentManager) GetOrCreateAgent(ctx context.Context, sessionID string, userName string) (*react.Agent, error) {
	return m.GetOrCreateAgentWithTools(ctx, sessionID, userName, nil, false)
}

// GetOrCreateAgentWithTools 获取或创建 Agent（支持自定义工具列表）
func (m *AgentManager) GetOrCreateAgentWithTools(ctx context.Context, sessionID string, userName string, toolNames []string, thinkingMode bool) (*react.Agent, error) {
	client := llm.GetLLMClient()
	if client == nil {
		return nil, fmt.Errorf("llm client not initialized")
	}

	model, modelName, err := client.GetModelForMode(ctx, thinkingMode)
	if err != nil {
		return nil, err
	}
	cacheKey := buildAgentCacheKey(sessionID, modelName)

	m.mu.RLock()
	if agent, ok := m.agents[cacheKey]; ok {
		m.mu.RUnlock()
		return agent, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if agent, ok := m.agents[cacheKey]; ok {
		return agent, nil
	}

	// 创建新 Agent
	agent, err := m.createAgentWithTools(ctx, userName, toolNames, model, modelName)
	if err != nil {
		return nil, err
	}

	m.agents[cacheKey] = agent
	return agent, nil
}

// createAgentWithTools 创建带指定工具的 React Agent
func (m *AgentManager) createAgentWithTools(ctx context.Context, userName string, toolNames []string, model eino_model.ToolCallingChatModel, modelName string) (*react.Agent, error) {
	if model == nil {
		return nil, fmt.Errorf("llm model not available")
	}

	// 1. 查询用户已索引的文件 ID
	fileIDs, err := file.GetIndexedFileIDsByUserName(userName)
	if err != nil {
		log.Printf("Warning: failed to get indexed file ids: %v", err)
		fileIDs = []uint{} // 继续执行，只是没有 RAG 工具
	}

	// 2. 如果没有指定工具，使用默认工具
	if len(toolNames) == 0 {
		toolNames = tools.GetToolRegistry().GetDefaultToolNames()
	}

	// 3. 从注册表获取工具
	registry := tools.GetToolRegistry()
	toolList, err := registry.GetToolsByNames(ctx, toolNames, fileIDs)
	if err != nil {
		return nil, fmt.Errorf("get tools failed: %w", err)
	}

	log.Printf("Creating agent with model=%s, tools=%v", modelName, toolNames)

	// 4. 创建 React Agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: toolList,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create react agent failed: %w", err)
	}

	return agent, nil
}

// RecreateAgentWithTools 重新创建带新工具的 Agent
func (m *AgentManager) RecreateAgentWithTools(ctx context.Context, sessionID string, userName string, toolNames []string, thinkingMode bool) (*react.Agent, error) {
	client := llm.GetLLMClient()
	if client == nil {
		return nil, fmt.Errorf("llm client not initialized")
	}

	model, modelName, err := client.GetModelForMode(ctx, thinkingMode)
	if err != nil {
		return nil, err
	}
	cacheKey := buildAgentCacheKey(sessionID, modelName)

	m.mu.Lock()
	defer m.mu.Unlock()

	// 删除旧 Agent
	delete(m.agents, cacheKey)

	// 创建新 Agent
	agent, err := m.createAgentWithTools(ctx, userName, toolNames, model, modelName)
	if err != nil {
		return nil, err
	}

	m.agents[cacheKey] = agent
	return agent, nil
}

// Generate 同步生成响应
func (m *AgentManager) Generate(ctx context.Context, sessionID string, userName string, messages []*schema.Message) (*schema.Message, error) {
	agent, err := m.GetOrCreateAgent(ctx, sessionID, userName)
	if err != nil {
		return nil, err
	}
	return agent.Generate(ctx, messages)
}

// GenerateWithTools 使用指定工具同步生成响应
func (m *AgentManager) GenerateWithTools(ctx context.Context, sessionID string, userName string, messages []*schema.Message, toolNames []string, thinkingMode bool) (*schema.Message, error) {
	agent, err := m.GetOrCreateAgentWithTools(ctx, sessionID, userName, toolNames, thinkingMode)
	if err != nil {
		return nil, err
	}
	return agent.Generate(ctx, messages)
}

// OpenStreamWithTools 获取底层流读取器，调用方负责 Close。
func (m *AgentManager) OpenStreamWithTools(ctx context.Context, sessionID string, userName string, messages []*schema.Message, toolNames []string, thinkingMode bool) (*schema.StreamReader[*schema.Message], error) {
	agent, err := m.GetOrCreateAgentWithTools(ctx, sessionID, userName, toolNames, thinkingMode)
	if err != nil {
		return nil, err
	}

	stream, err := agent.Stream(ctx, messages)
	if err != nil {
		return nil, err
	}

	return stream, nil
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
