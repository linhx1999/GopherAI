package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/tool/sequentialthinking"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	sequentialThinkingToolName        = "sequential_thinking"
	sequentialThinkingToolDisplayName = "逐步思考"
	sequentialThinkingToolDescription = "使用逐步思考流程拆解复杂问题，输出结构化的思考过程和中间结论。"
)

type sequentialThinkingTool struct {
	inner tool.InvokableTool
}

func (t *sequentialThinkingTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: sequentialThinkingToolName,
		Desc: sequentialThinkingToolDescription,
	}, nil
}

func (t *sequentialThinkingTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	if t.inner == nil {
		return "", fmt.Errorf("%s tool not initialized", sequentialThinkingToolName)
	}
	return t.inner.InvokableRun(ctx, argumentsInJSON, opts...)
}

func sequentialThinkingDescriptor() localToolDescriptor {
	return localToolDescriptor{
		Name:        sequentialThinkingToolName,
		DisplayName: sequentialThinkingToolDisplayName,
		Description: sequentialThinkingToolDescription,
		Parameters:  map[string]interface{}{},
		Category:    "builtin",
		Build: func(ctx context.Context, fileRefIDs []uint) (tool.BaseTool, error) {
			builtinTool, err := sequentialthinking.NewTool()
			if err != nil {
				return nil, fmt.Errorf("create %s tool failed: %w", sequentialThinkingToolName, err)
			}
			return &sequentialThinkingTool{inner: builtinTool}, nil
		},
	}
}

func GetSequentialThinkingTool() (tool.BaseTool, error) {
	inner, err := sequentialthinking.NewTool()
	if err != nil {
		return nil, fmt.Errorf("create %s tool failed: %w", sequentialThinkingToolName, err)
	}
	return &sequentialThinkingTool{inner: inner}, nil
}
