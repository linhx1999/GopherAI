package agent

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/cloudwego/eino/adk"
	eino_model "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"

	"GopherAI/common/llm"
)

// AgentManager Agent 管理器
type AgentManager struct{}

type AgentSessionConfig struct {
	Instruction     string
	Model           eino_model.ToolCallingChatModel
	ModelName       string
	ToolsNodeConfig compose.ToolsNodeConfig
}

const (
	chatAgentName        = "chat_agent"
	chatAgentDescription = "单 Agent 聊天助手，负责普通问答、工具调用与知识检索。"
)

var (
	manager     *AgentManager
	managerOnce sync.Once
)

// GetAgentManager 获取全局 Agent 管理器
func GetAgentManager() *AgentManager {
	managerOnce.Do(func() {
		manager = &AgentManager{}
	})
	return manager
}

func (m *AgentManager) resolveAgentSessionConfig(ctx context.Context, toolsNodeConfig compose.ToolsNodeConfig, thinkingMode bool, instruction string) (*AgentSessionConfig, error) {
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

	return &AgentSessionConfig{
		Instruction:     instruction,
		Model:           model,
		ModelName:       modelName,
		ToolsNodeConfig: toolsNodeConfig,
	}, nil
}

func (m *AgentManager) buildAgent(ctx context.Context, config *AgentSessionConfig) (adk.Agent, error) {
	if config == nil || config.Model == nil {
		return nil, fmt.Errorf("agent config not available")
	}

	log.Printf("Creating ADK chat agent with model=%s, tool_count=%d", config.ModelName, len(config.ToolsNodeConfig.Tools))

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        chatAgentName,
		Description: chatAgentDescription,
		Instruction: config.Instruction,
		Model:       config.Model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: config.ToolsNodeConfig,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create adk chat model agent failed: %w", err)
	}

	return agent, nil
}

// CreateAgentForChat 基于当前请求创建一个新的 ChatModelAgent。
func (m *AgentManager) CreateAgentForChat(ctx context.Context, toolsNodeConfig compose.ToolsNodeConfig, thinkingMode bool, instruction string) (adk.Agent, error) {
	config, err := m.resolveAgentSessionConfig(ctx, toolsNodeConfig, thinkingMode, instruction)
	if err != nil {
		return nil, err
	}
	return m.buildAgent(ctx, config)
}
