package tools

import (
	"context"
	"log"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino-ext/components/tool/sequentialthinking"
	mcpClient "github.com/mark3labs/mcp-go/client"
	mcpschema "github.com/mark3labs/mcp-go/mcp"
)

// ToolInfo 工具信息（用于 API 返回）
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Category    string                 `json:"category"` // "builtin" | "mcp" | "rag"
}

// MCPClientConfig MCP 客户端配置
type MCPClientConfig struct {
	Name string // 客户端名称
	URL  string // MCP 服务器 URL
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	mu          sync.RWMutex
	builtin     map[string]tool.BaseTool           // 内置工具
	mcpClients  map[string]mcpClient.MCPClient     // MCP 客户端
	mcpConfigs  map[string]*MCPClientConfig        // MCP 配置
}

var (
	registry     *ToolRegistry
	registryOnce sync.Once
)

// GetToolRegistry 获取全局工具注册表
func GetToolRegistry() *ToolRegistry {
	registryOnce.Do(func() {
		registry = &ToolRegistry{
			builtin:    make(map[string]tool.BaseTool),
			mcpClients: make(map[string]mcpClient.MCPClient),
			mcpConfigs: make(map[string]*MCPClientConfig),
		}
		registry.initBuiltinTools()
	})
	return registry
}

// initBuiltinTools 初始化内置工具
func (r *ToolRegistry) initBuiltinTools() {
	// 注册 Sequential Thinking 工具（使用官方实现）
	tool, err := sequentialthinking.NewTool()
	if err != nil {
		log.Printf("Failed to create sequential_thinking tool: %v", err)
	} else {
		r.builtin["sequential_thinking"] = tool
	}
}

// RegisterMCPClient 注册 MCP 客户端配置
func (r *ToolRegistry) RegisterMCPClient(name string, url string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mcpConfigs[name] = &MCPClientConfig{
		Name: name,
		URL:  url,
	}
}

// RegisterMCPClientInstance 注册 MCP 客户端实例
func (r *ToolRegistry) RegisterMCPClientInstance(name string, client mcpClient.MCPClient) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mcpClients[name] = client
}

// UnregisterMCPClient 注销 MCP 客户端
func (r *ToolRegistry) UnregisterMCPClient(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if client, ok := r.mcpClients[name]; ok {
		client.Close()
		delete(r.mcpClients, name)
	}
	delete(r.mcpConfigs, name)
}

// GetBuiltinTool 获取内置工具
func (r *ToolRegistry) GetBuiltinTool(name string) tool.BaseTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.builtin[name]
}

// initMCPClient 初始化 MCP 客户端
func (r *ToolRegistry) initMCPClient(ctx context.Context, name string) (mcpClient.MCPClient, error) {
	r.mu.RLock()
	config, ok := r.mcpConfigs[name]
	r.mu.RUnlock()

	if !ok {
		return nil, nil
	}

	// 检查是否已初始化
	r.mu.RLock()
	if client, ok := r.mcpClients[name]; ok {
		r.mu.RUnlock()
		return client, nil
	}
	r.mu.RUnlock()

	// 创建 SSE MCP 客户端
	cli, err := mcpClient.NewSSEMCPClient(config.URL)
	if err != nil {
		return nil, err
	}

	// 启动客户端
	if err := cli.Start(ctx); err != nil {
		return nil, err
	}

	// 初始化
	initRequest := mcpschema.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcpschema.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcpschema.Implementation{
		Name:    "gopherai-client",
		Version: "1.0.0",
	}
	if _, err := cli.Initialize(ctx, initRequest); err != nil {
		cli.Close()
		return nil, err
	}

	// 缓存客户端
	r.mu.Lock()
	r.mcpClients[name] = cli
	r.mu.Unlock()

	return cli, nil
}

// GetToolsByNames 根据名称列表获取工具实例
func (r *ToolRegistry) GetToolsByNames(ctx context.Context, names []string, ragFileIDs []uint) ([]tool.BaseTool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tool.BaseTool, 0, len(names))
	seen := make(map[string]struct{})

	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		switch name {
		case "knowledge_search":
			// RAG 工具需要动态创建
			if len(ragFileIDs) > 0 {
				result = append(result, NewRAGTool(ragFileIDs))
			}

		case "sequential_thinking":
			// 内置工具
			if t, ok := r.builtin["sequential_thinking"]; ok {
				result = append(result, t)
			}

		default:
			// 尝试从 MCP 客户端获取
			for clientName := range r.mcpConfigs {
				cli, err := r.initMCPClient(ctx, clientName)
				if err != nil {
					log.Printf("Failed to init MCP client %s: %v", clientName, err)
					continue
				}

				tools, err := einomcp.GetTools(ctx, &einomcp.Config{
					Cli:          cli,
					ToolNameList: []string{name},
				})
				if err != nil {
					log.Printf("Failed to get MCP tools from %s: %v", clientName, err)
					continue
				}

				for _, t := range tools {
					info, _ := t.Info(ctx)
					if info != nil && info.Name == name {
						result = append(result, t)
						break
					}
				}
			}
		}
	}

	return result, nil
}

// ListAvailableTools 列出所有可用工具
func (r *ToolRegistry) ListAvailableTools(ctx context.Context) []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ToolInfo, 0)

	// 1. 内置工具
	for _, t := range r.builtin {
		info, err := t.Info(ctx)
		if err != nil {
			continue
		}
		result = append(result, ToolInfo{
			Name:        info.Name,
			Description: info.Desc,
			Parameters:  extractParams(info),
			Category:    "builtin",
		})
	}

	// 2. 固定添加 knowledge_search（RAG 工具）
	result = append(result, ToolInfo{
		Name:        "knowledge_search",
		Description: "从知识库中检索相关文档。当用户问题涉及已上传的文档内容时，使用此工具获取相关信息。",
		Parameters: map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "检索查询语句",
				"required":    true,
			},
			"top_k": map[string]interface{}{
				"type":        "number",
				"description": "返回的相关文档数量，默认为 5",
				"required":    false,
			},
		},
		Category: "rag",
	})

	// 3. MCP 工具（异步加载，避免阻塞）
	for clientName := range r.mcpConfigs {
		cli, err := r.initMCPClient(ctx, clientName)
		if err != nil {
			continue
		}

		tools, err := einomcp.GetTools(ctx, &einomcp.Config{Cli: cli})
		if err != nil {
			continue
		}

		for _, t := range tools {
			info, err := t.Info(ctx)
			if err != nil {
				continue
			}
			result = append(result, ToolInfo{
				Name:        info.Name,
				Description: info.Desc,
				Parameters:  extractParams(info),
				Category:    "mcp",
			})
		}
	}

	return result
}

// extractParams 从 ToolInfo 提取参数定义
func extractParams(info *schema.ToolInfo) map[string]interface{} {
	params := make(map[string]interface{})

	if info.ParamsOneOf == nil {
		return params
	}

	// 尝试使用 ByParams 方式获取参数
	// 注意：ParamsOneOf 的内部字段是未导出的，我们只能通过其他方式获取
	// 这里返回一个空 map，实际参数需要从工具文档中获取
	return params
}

// GetDefaultToolNames 获取默认工具名称列表
func (r *ToolRegistry) GetDefaultToolNames() []string {
	return []string{"knowledge_search"}
}

// HasTool 检查工具是否存在
func (r *ToolRegistry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 检查内置工具
	if _, ok := r.builtin[name]; ok {
		return true
	}

	// 检查特殊工具
	if name == "knowledge_search" {
		return true
	}

	// 检查 MCP 工具
	return len(r.mcpConfigs) > 0
}