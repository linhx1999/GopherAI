package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"GopherAI/common/rag"
)

const (
	knowledgeSearchToolName        = "knowledge_search"
	knowledgeSearchToolDisplayName = "知识库检索"
	knowledgeSearchToolDescription = "从知识库中检索相关文档。当用户问题涉及已上传的文档内容时，使用此工具获取相关信息。"
)

var KnowledgeSearchTool = Tool{
	name:        knowledgeSearchToolName,
	displayName: knowledgeSearchToolDisplayName,
	description: knowledgeSearchToolDescription,
	tool:        newKnowledgeSearchTool(),
}

// RAGTool RAG 检索工具
type RAGTool struct{}

// RAGToolParams RAG 工具参数
type RAGToolParams struct {
	Query string `json:"query"` // 检索查询
	TopK  int    `json:"top_k"` // 返回文档数量（可选）
}

// RAGToolResult RAG 工具返回结果
type RAGToolResult struct {
	Documents []string `json:"documents"` // 检索到的文档内容
	Count     int      `json:"count"`     // 文档数量
}

// Info 返回工具元信息
func (t *RAGTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: knowledgeSearchToolName,
		Desc: knowledgeSearchToolDescription,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {
				Type:     "string",
				Desc:     "检索查询语句",
				Required: true,
			},
			"top_k": {
				Type:     "number",
				Desc:     "返回的相关文档数量，默认为 5",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun 执行 RAG 检索
func (t *RAGTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	// 1. 解析参数
	var params RAGToolParams
	if err := json.Unmarshal([]byte(argumentsInJSON), &params); err != nil {
		return "", fmt.Errorf("parse arguments failed: %w", err)
	}

	// 2. 验证参数
	if params.Query == "" {
		return "", fmt.Errorf("query is required")
	}

	// 3. 设置默认值
	if params.TopK == 0 {
		params.TopK = 5
	}

	// 4. 执行检索
	docs, err := rag.RetrieveDocuments(ctx, params.Query, params.TopK)
	if err != nil {
		return "", fmt.Errorf("rag retrieve failed: %w", err)
	}

	// 5. 构建结果
	result := RAGToolResult{
		Documents: make([]string, 0, len(docs)),
		Count:     len(docs),
	}
	for _, doc := range docs {
		result.Documents = append(result.Documents, doc.Content)
	}

	// 6. 序列化返回
	bytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func newKnowledgeSearchTool() tool.InvokableTool {
	return &RAGTool{}
}

func GetKnowledgeSearchTool() tool.BaseTool {
	return newKnowledgeSearchTool()
}
