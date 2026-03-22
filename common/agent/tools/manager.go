package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
)

type Tool struct {
	name        string
	displayName string
	description string
	tool        tool.BaseTool
}

// UnknownToolError 表示请求中包含不存在的工具名。
type UnknownToolError struct {
	Names []string
}

func (e *UnknownToolError) Error() string {
	return fmt.Sprintf("unknown tools: %s", strings.Join(e.Names, ", "))
}

var BuiltInTools []Tool

func init() {
	BuiltInTools = []Tool{
		KnowledgeSearchTool,
		SequentialThinkingTool,
	}
}

func isUnknownToolError(err error) bool {
	var target *UnknownToolError
	return errors.As(err, &target)
}

func normalizeToolNames(requestedNames []string) ([]string, []string) {
	normalized := make([]string, 0, len(requestedNames))
	seen := make(map[string]struct{}, len(requestedNames))
	unknown := make([]string, 0)

	for _, rawName := range requestedNames {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}

		if !isBuiltInTool(name) {
			unknown = append(unknown, name)
			continue
		}

		normalized = append(normalized, name)
	}

	return normalized, unknown
}

func isBuiltInTool(name string) bool {
	for _, builtInTool := range BuiltInTools {
		if builtInTool.name == name {
			return true
		}
	}
	return false
}

func buildResolvedTool(_ context.Context, name string) (tool.BaseTool, error) {
	switch name {
	case knowledgeSearchToolName:
		return GetKnowledgeSearchTool(), nil
	case SequentialThinkingToolName():
		return GetSequentialThinkingTool()
	default:
		return nil, &UnknownToolError{Names: []string{name}}
	}
}

func NormalizeToolNames(requestedNames []string) []string {
	normalized, _ := normalizeToolNames(requestedNames)
	return normalized
}

func ResolveRequestedTools(ctx context.Context, requestedNames []string) ([]tool.BaseTool, error) {
	normalizedNames, unknownNames := normalizeToolNames(requestedNames)
	if len(unknownNames) > 0 {
		return nil, &UnknownToolError{Names: unknownNames}
	}

	resolvedTools := make([]tool.BaseTool, 0, len(normalizedNames))
	for _, name := range normalizedNames {
		resolvedTool, err := buildResolvedTool(ctx, name)
		if err != nil {
			return nil, err
		}
		resolvedTools = append(resolvedTools, resolvedTool)
	}

	return resolvedTools, nil
}
