package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"GopherAI/common/agent/tools"
	"GopherAI/common/llm"
	"GopherAI/dao/file"
	"GopherAI/model"
)

// StreamCallback 流式回调函数
type StreamCallback func(content string)

// ToolCallCallback 工具调用回调函数
type ToolCallCallback func(toolCall model.ToolCall)

// AgentResult Agent 执行结果
type AgentResult struct {
	SessionID    string           `json:"session_id"`
	MessageIndex int              `json:"message_index"`
	Role         string           `json:"role"`
	Content      string           `json:"content"`
	ToolCalls    []model.ToolCall `json:"tool_calls,omitempty"`
}

// AgentManager Agent 管理器
type AgentManager struct {
	mu     sync.RWMutex
	agents map[string]*react.Agent // sessionID -> Agent
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

// GetOrCreateAgent 获取或创建 Agent（根据用户名动态查询文件ID，使用默认工具）
func (m *AgentManager) GetOrCreateAgent(ctx context.Context, sessionID string, userName string) (*react.Agent, error) {
	return m.GetOrCreateAgentWithTools(ctx, sessionID, userName, nil)
}

// GetOrCreateAgentWithTools 获取或创建 Agent（支持自定义工具列表）
func (m *AgentManager) GetOrCreateAgentWithTools(ctx context.Context, sessionID string, userName string, toolNames []string) (*react.Agent, error) {
	m.mu.RLock()
	if agent, ok := m.agents[sessionID]; ok {
		m.mu.RUnlock()
		return agent, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if agent, ok := m.agents[sessionID]; ok {
		return agent, nil
	}

	// 创建新 Agent
	agent, err := m.createAgentWithTools(ctx, userName, toolNames)
	if err != nil {
		return nil, err
	}

	m.agents[sessionID] = agent
	return agent, nil
}

// createAgentWithTools 创建带指定工具的 React Agent
func (m *AgentManager) createAgentWithTools(ctx context.Context, userName string, toolNames []string) (*react.Agent, error) {
	// 1. 获取 LLM 客户端
	client := llm.GetLLMClient()
	if client == nil {
		return nil, fmt.Errorf("llm client not initialized")
	}

	// 2. 获取底层模型（需要 ToolCallingChatModel）
	model := client.GetModel()
	if model == nil {
		return nil, fmt.Errorf("llm model not available")
	}

	// 3. 查询用户已索引的文件 ID
	fileIDs, err := file.GetIndexedFileIDsByUserName(userName)
	if err != nil {
		log.Printf("Warning: failed to get indexed file ids: %v", err)
		fileIDs = []uint{} // 继续执行，只是没有 RAG 工具
	}

	// 4. 如果没有指定工具，使用默认工具
	if len(toolNames) == 0 {
		toolNames = tools.GetToolRegistry().GetDefaultToolNames()
	}

	// 5. 从注册表获取工具
	registry := tools.GetToolRegistry()
	toolList, err := registry.GetToolsByNames(ctx, toolNames, fileIDs)
	if err != nil {
		return nil, fmt.Errorf("get tools failed: %w", err)
	}

	log.Printf("Creating agent with %d tools: %v", len(toolList), toolNames)

	// 6. 创建 React Agent
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
func (m *AgentManager) RecreateAgentWithTools(ctx context.Context, sessionID string, userName string, toolNames []string) (*react.Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 删除旧 Agent
	delete(m.agents, sessionID)

	// 创建新 Agent
	agent, err := m.createAgentWithTools(ctx, userName, toolNames)
	if err != nil {
		return nil, err
	}

	m.agents[sessionID] = agent
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
func (m *AgentManager) GenerateWithTools(ctx context.Context, sessionID string, userName string, messages []*schema.Message, toolNames []string) (*schema.Message, error) {
	agent, err := m.GetOrCreateAgentWithTools(ctx, sessionID, userName, toolNames)
	if err != nil {
		return nil, err
	}
	return agent.Generate(ctx, messages)
}

// Stream 流式生成响应
func (m *AgentManager) Stream(ctx context.Context, sessionID string, userName string, messages []*schema.Message, cb StreamCallback) (string, error) {
	agent, err := m.GetOrCreateAgent(ctx, sessionID, userName)
	if err != nil {
		return "", err
	}

	stream, err := agent.Stream(ctx, messages)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var fullResp string
	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fullResp, err
		}
		if msg.Content != "" {
			fullResp += msg.Content
			if cb != nil {
				cb(msg.Content)
			}
		}
	}
	return fullResp, nil
}

// StreamWithCallbacks 流式生成响应（支持工具调用回调）
func (m *AgentManager) StreamWithCallbacks(ctx context.Context, sessionID string, userName string, messages []*schema.Message, toolNames []string, contentCb StreamCallback, toolCallCb ToolCallCallback) (string, []model.ToolCall, error) {
	agent, err := m.GetOrCreateAgentWithTools(ctx, sessionID, userName, toolNames)
	if err != nil {
		return "", nil, err
	}

	stream, err := agent.Stream(ctx, messages)
	if err != nil {
		return "", nil, err
	}
	defer stream.Close()

	var fullResp string
	var toolCalls []model.ToolCall

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fullResp, toolCalls, err
		}

		// 处理内容
		if msg.Content != "" {
			fullResp += msg.Content
			if contentCb != nil {
				contentCb(msg.Content)
			}
		}

		// 处理工具调用
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				toolCall := model.ToolCall{
					ToolID:    tc.ID,
					Function:  tc.Function.Name,
					Arguments: json.RawMessage(tc.Function.Arguments),
				}
				toolCalls = append(toolCalls, toolCall)
				if toolCallCb != nil {
					toolCallCb(toolCall)
				}
			}
		}
	}
	return fullResp, toolCalls, nil
}

// ClearAgent 清除指定会话的 Agent 缓存
func (m *AgentManager) ClearAgent(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, sessionID)
}

// ParseToolCalls 从 JSON 解析工具调用
func ParseToolCalls(data json.RawMessage) []model.ToolCall {
	if data == nil {
		return nil
	}
	var calls []model.ToolCall
	json.Unmarshal(data, &calls)
	return calls
}

// GetCurrentTimestamp 获取当前时间戳
func GetCurrentTimestamp() time.Time {
	return time.Now()
}