package tools

import (
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
		SequentialThinkingTool,
		knowledgeSearchToolDefinition,
	}
}

func isUnknownToolError(err error) bool {
	var target *UnknownToolError
	return errors.As(err, &target)
}
