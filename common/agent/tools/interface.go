package tools

// ToolInfo 工具信息（用于 API 返回）
type ToolInfo struct {
	Name        string `json:"name"`         // API 调用名
	DisplayName string `json:"display_name"` // 前端展示名
	Description string `json:"description"`
}

func ListAvailableTools() []ToolInfo {
	result := make([]ToolInfo, 0, len(BuiltInTools))
	for _, tool := range BuiltInTools {
		result = append(result, ToolInfo{
			Name:        tool.name,
			DisplayName: tool.displayName,
			Description: tool.description,
		})
	}

	return result
}

func IsUnknownToolError(err error) bool {
	return isUnknownToolError(err)
}
