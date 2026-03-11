package tools

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

// ToolInfo 工具信息（用于 API 返回）
type ToolInfo struct {
	Name        string                 `json:"name"`         // API 调用名
	DisplayName string                 `json:"display_name"` // 前端展示名
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Category    string                 `json:"category"` // "builtin" | "rag"
}

type localToolDescriptor struct {
	Name        string
	DisplayName string
	Description string
	Parameters  map[string]interface{}
	Category    string
	Build       func(ctx context.Context, fileRefIDs []uint) (tool.BaseTool, error)
}

// UnknownToolError 表示请求中包含不存在的工具名。
type UnknownToolError struct {
	Names []string
}

func (e *UnknownToolError) Error() string {
	return fmt.Sprintf("unknown tools: %s", strings.Join(e.Names, ", "))
}

var localToolDescriptors = map[string]localToolDescriptor{
	knowledgeSearchToolName:    knowledgeSearchDescriptor(),
	sequentialThinkingToolName: sequentialThinkingDescriptor(),
}

func NormalizeToolNames(names []string) []string {
	normalized := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))

	for _, name := range names {
		trimmedName := strings.TrimSpace(name)
		if trimmedName == "" {
			continue
		}
		if _, ok := seen[trimmedName]; ok {
			continue
		}
		seen[trimmedName] = struct{}{}
		normalized = append(normalized, trimmedName)
	}

	return normalized
}

// BuildRequestedTools 根据请求中的工具名构建工具实例列表。
func BuildRequestedTools(ctx context.Context, names []string, fileRefIDs []uint) ([]tool.BaseTool, error) {
	normalizedNames := NormalizeToolNames(names)
	tools := make([]tool.BaseTool, 0, len(normalizedNames))
	unknownTools := make([]string, 0)

	for _, name := range normalizedNames {
		descriptor, ok := localToolDescriptors[name]
		if !ok {
			unknownTools = append(unknownTools, name)
			continue
		}

		builtTool, err := descriptor.Build(ctx, fileRefIDs)
		if err != nil {
			return nil, err
		}
		if builtTool != nil {
			tools = append(tools, builtTool)
		}
	}

	if len(unknownTools) > 0 {
		return nil, &UnknownToolError{Names: unknownTools}
	}

	return tools, nil
}

// ListAvailableTools 返回当前全局工具 map 中的工具目录。
func ListAvailableTools() []ToolInfo {
	result := make([]ToolInfo, 0, len(localToolDescriptors))
	for _, descriptor := range localToolDescriptors {
		result = append(result, ToolInfo{
			Name:        descriptor.Name,
			DisplayName: descriptor.DisplayName,
			Description: descriptor.Description,
			Parameters:  cloneParameters(descriptor.Parameters),
			Category:    descriptor.Category,
		})
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

func IsUnknownToolError(err error) bool {
	var target *UnknownToolError
	return errors.As(err, &target)
}
