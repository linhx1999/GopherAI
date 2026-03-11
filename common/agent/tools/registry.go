package tools

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"

	einomcp "github.com/cloudwego/eino-ext/components/tool/mcp"
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

type localToolDescriptor struct {
	Name        string
	DisplayName string
	Description string
	Parameters  map[string]interface{}
	Category    string
	Build       func(ctx context.Context, fileRefIDs []uint) (tool.BaseTool, bool, error)
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	mu         sync.RWMutex
	localTools map[string]localToolDescriptor
	mcpClients map[string]mcpClient.MCPClient
	mcpConfigs map[string]*MCPClientConfig
}

var (
	registry     *ToolRegistry
	registryOnce sync.Once
)

// GetToolRegistry 获取全局工具注册表
func GetToolRegistry() *ToolRegistry {
	registryOnce.Do(func() {
		registry = &ToolRegistry{
			localTools: make(map[string]localToolDescriptor),
			mcpClients: make(map[string]mcpClient.MCPClient),
			mcpConfigs: make(map[string]*MCPClientConfig),
		}
		registry.initLocalTools()
	})
	return registry
}

func (r *ToolRegistry) initLocalTools() {
	for _, descriptor := range []localToolDescriptor{
		knowledgeSearchDescriptor(),
		sequentialThinkingDescriptor(),
	} {
		r.registerLocalTool(descriptor)
	}
}

func (r *ToolRegistry) registerLocalTool(descriptor localToolDescriptor) {
	r.localTools[descriptor.Name] = descriptor
}

func (r *ToolRegistry) getLocalToolDescriptor(name string) (localToolDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	descriptor, ok := r.localTools[name]
	return descriptor, ok
}

func (r *ToolRegistry) listLocalToolDescriptors() []localToolDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]localToolDescriptor, 0, len(r.localTools))
	for _, descriptor := range r.localTools {
		result = append(result, descriptor)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return result[i].Category < result[j].Category
		}
		return result[i].DisplayName < result[j].DisplayName
	})

	return result
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

	r.mu.RLock()
	if client, ok := r.mcpClients[name]; ok {
		r.mu.RUnlock()
		return client, nil
	}
	r.mu.RUnlock()

	cli, err := mcpClient.NewSSEMCPClient(config.URL)
	if err != nil {
		return nil, err
	}

	if err := cli.Start(ctx); err != nil {
		return nil, err
	}

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
	case legacySequentialThinkingToolName:
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
func (r *ToolRegistry) ResolveTools(ctx context.Context, names []string, fileRefIDs []uint) ([]tool.BaseTool, error) {
	normalizedNames := NormalizeToolNames(names)
	resolvedTools := make([]tool.BaseTool, 0, len(normalizedNames))

	for _, toolName := range normalizedNames {
		if descriptor, ok := r.getLocalToolDescriptor(toolName); ok {
			resolvedTool, enabled, err := descriptor.Build(ctx, fileRefIDs)
			if err != nil {
				return nil, err
			}
			if enabled && resolvedTool != nil {
				resolvedTools = append(resolvedTools, resolvedTool)
			}
			continue
		}

		if mcpTool, ok := r.resolveMCPTool(ctx, toolName); ok {
			resolvedTools = append(resolvedTools, mcpTool)
		}
	}

	return resolvedTools, nil
}

// ListAvailableTools 列出所有可用工具
func (r *ToolRegistry) ListAvailableTools(ctx context.Context) []ToolInfo {
	result := make([]ToolInfo, 0)

	for _, descriptor := range r.listLocalToolDescriptors() {
		result = append(result, ToolInfo{
			Name:        descriptor.Name,
			DisplayName: descriptor.DisplayName,
			Description: descriptor.Description,
			Parameters:  cloneParameters(descriptor.Parameters),
			Category:    descriptor.Category,
		})
	}

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
				DisplayName: info.Name,
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

func cloneParameters(params map[string]interface{}) map[string]interface{} {
	if len(params) == 0 {
		return map[string]interface{}{}
	}

	cloned := make(map[string]interface{}, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}

// extractParams 从 ToolInfo 提取参数定义
func extractParams(info *schema.ToolInfo) map[string]interface{} {
	if info == nil || info.ParamsOneOf == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{}
}

// HasTool 检查工具是否存在
func (r *ToolRegistry) HasTool(name string) bool {
	name = normalizeToolName(name)

	if _, ok := r.getLocalToolDescriptor(name); ok {
		return true
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.mcpConfigs) > 0
}
