package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino-ext/components/tool/sequentialthinking"
	"github.com/cloudwego/eino/components/tool"
)

const (
	sequentialThinkingDisplayName = "逐步思考"
	sequentialThinkingDescription = "使用逐步思考流程拆解复杂问题，输出结构化的思考过程和中间结论。"
)

var SequentialThinkingTool = initSequentialThinking()

func initSequentialThinking() Tool {
	builtTool, err := sequentialthinking.NewTool()
	if err != nil {
		panic(fmt.Sprintf("create sequentialthinking tool definition failed: %v", err))
	}

	info, err := builtTool.Info(context.Background())
	if err != nil {
		panic(fmt.Sprintf("load sequentialthinking tool info failed: %v", err))
	}
	if info == nil {
		panic("load sequentialthinking tool info failed: nil info")
	}

	toolName := strings.TrimSpace(info.Name)
	if toolName == "" {
		panic("load sequentialthinking tool info failed: empty name")
	}

	return Tool{
		name:        toolName,
		displayName: sequentialThinkingDisplayName,
		description: sequentialThinkingDescription,
		tool:        builtTool,
	}
}

func SequentialThinkingToolName() string {
	return SequentialThinkingTool.name
}

func GetSequentialThinkingTool() (tool.BaseTool, error) {
	return sequentialthinking.NewTool()
}
