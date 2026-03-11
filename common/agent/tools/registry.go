package tools

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino-ext/components/tool/sequentialthinking"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	mcpClient "github.com/mark3labs/mcp-go/client"
	mcpschema "github.com/mark3labs/mcp-go/mcp"
)

// ToolInfo 工具信息（用于 API 返回）
type ToolInfo struct {
	Name        string                 `json:"name"`         // API 调用名
	DisplayName string                 `json:"display_name"` // 前端展示名
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
	mu         sync.RWMutex
	builtin    map[string]tool.BaseTool       // 内置工具
	mcpClients map[string]mcpClient.MCPClient // MCP 客户端
	mcpConfigs map[string]*MCPClientConfig    // MCP 配置
}

var (
	registry     *ToolRegistry
	registryOnce sync.Once
)

const (
	sequentialThinkingToolName      = "sequentialthinking"
	legacySequentialThinkingToolKey = "sequential_thinking"
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
		r.builtin[sequentialThinkingToolName] = tool
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

func (r *ToolRegistry) listBuiltinTools() []tool.BaseTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]tool.BaseTool, 0, len(r.builtin))
	for _, builtinTool := range r.builtin {
		result = append(result, builtinTool)
	}
	return result
}

func (r *ToolRegistry) listMCPClientNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.mcpConfigs))
	for clientName := range r.mcpConfigs {
		names = append(names, clientName)
	}
	sort.Strings(names)
	return names
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

func NormalizeToolNames(names []string) []string {
	normalized := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))

	for _, name := range names {
		trimmedName := normalizeToolName(name)
		if trimmedName == "" {
			continue
		}
		if _, ok := seen[trimmedName]; ok {
			continue
		}
		seen[trimmedName] = struct{}{}
		normalized = append(normalized, trimmedName)
	}

	sort.Strings(normalized)
	return normalized
}

func normalizeToolName(name string) string {
	trimmedName := strings.TrimSpace(name)
	switch trimmedName {
	case legacySequentialThinkingToolKey:
		return sequentialThinkingToolName
	default:
		return trimmedName
	}
}

func (r *ToolRegistry) resolveMCPTool(ctx context.Context, toolName string) (tool.BaseTool, bool) {
	for _, clientName := range r.listMCPClientNames() {
		cli, err := r.initMCPClient(ctx, clientName)
		if err != nil {
			log.Printf("Failed to init MCP client %s: %v", clientName, err)
			continue
		}

		mcpTools, err := einomcp.GetTools(ctx, &einomcp.Config{
			Cli:          cli,
			ToolNameList: []string{toolName},
		})
		if err != nil {
			log.Printf("Failed to get MCP tools from %s: %v", clientName, err)
			continue
		}

		for _, resolvedTool := range mcpTools {
			info, _ := resolvedTool.Info(ctx)
			if info != nil && info.Name == toolName {
				return resolvedTool, true
			}
		}
	}

	return nil, false
}

// ResolveTools 根据 API 调用名列表解析工具实例。
func (r *ToolRegistry) ResolveTools(ctx context.Context, names []string, ragFileIDs []uint) ([]tool.BaseTool, error) {
	normalizedNames := NormalizeToolNames(names)
	resolvedTools := make([]tool.BaseTool, 0, len(normalizedNames))

	for _, toolName := range normalizedNames {
		switch toolName {
		case "knowledge_search":
			if len(ragFileIDs) > 0 {
				resolvedTools = append(resolvedTools, NewRAGTool(ragFileIDs))
			}
		case sequentialThinkingToolName:
			if builtinTool := r.GetBuiltinTool(sequentialThinkingToolName); builtinTool != nil {
				resolvedTools = append(resolvedTools, builtinTool)
			}
		default:
			if mcpTool, ok := r.resolveMCPTool(ctx, toolName); ok {
				resolvedTools = append(resolvedTools, mcpTool)
			}
		}
	}

	return resolvedTools, nil
}

// ListAvailableTools 列出所有可用工具
func (r *ToolRegistry) ListAvailableTools(ctx context.Context) []ToolInfo {
	result := make([]ToolInfo, 0)

	// 1. 内置工具
	for _, builtinTool := range r.listBuiltinTools() {
		info, err := builtinTool.Info(ctx)
		if err != nil {
			continue
		}
		result = append(result, ToolInfo{
			Name:        info.Name,
			DisplayName: toolDisplayName(info.Name),
			Description: info.Desc,
			Parameters:  extractParams(info),
			Category:    "builtin",
		})
	}

	// 2. 固定添加 knowledge_search（RAG 工具）
	result = append(result, ToolInfo{
		Name:        "knowledge_search",
		DisplayName: toolDisplayName("knowledge_search"),
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
	for _, clientName := range r.listMCPClientNames() {
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
				DisplayName: toolDisplayName(info.Name),
				Description: info.Desc,
				Parameters:  extractParams(info),
				Category:    "mcp",
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].DisplayName < result[j].DisplayName
	})

	return result
}

func toolDisplayName(toolName string) string {
	switch normalizeToolName(toolName) {
	case "knowledge_search":
		return "知识库检索"
	case sequentialThinkingToolName:
		return "逐步思考"
	default:
		return toolName
	}
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

// HasTool 检查工具是否存在
func (r *ToolRegistry) HasTool(name string) bool {
	name = normalizeToolName(name)
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
