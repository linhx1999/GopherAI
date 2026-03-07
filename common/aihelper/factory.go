package aihelper

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// ModelCreator 定义模型创建函数类型（需要 context）
type ModelCreator func(ctx context.Context, config map[string]interface{}) (AIModel, error)

// ParseModelType 解析模型类型和文件 ID
// 返回格式：modelType, fileID
// 例如："2-1" -> "2", 1
func ParseModelType(modelType string) (string, uint) {
	parts := strings.Split(modelType, "-")
	if len(parts) == 2 {
		fileID, err := strconv.ParseUint(parts[1], 10, 32)
		if err == nil {
			return parts[0], uint(fileID)
		}
	}
	return modelType, 0
}

// AIModelFactory AI模型工厂
type AIModelFactory struct {
	creators map[string]ModelCreator
}

var (
	globalFactory *AIModelFactory
	factoryOnce   sync.Once
)

// GetGlobalFactory 获取全局单例
func GetGlobalFactory() *AIModelFactory {
	factoryOnce.Do(func() {
		globalFactory = &AIModelFactory{
			creators: make(map[string]ModelCreator),
		}
		globalFactory.registerCreators()
	})
	return globalFactory
}

// 注册模型
func (f *AIModelFactory) registerCreators() {
	//OpenAI
	f.creators["1"] = func(ctx context.Context, config map[string]interface{}) (AIModel, error) {
		return NewOpenAIModel(ctx)
	}

	// 阿里百炼 RAG 模型
	f.creators["2"] = func(ctx context.Context, config map[string]interface{}) (AIModel, error) {
		username, ok := config["username"].(string)
		if !ok {
			return nil, fmt.Errorf("RAG model requires username")
		}

		// 从 config 中获取文件 ID
		fileID := uint(0)
		if fid, ok := config["fileID"].(uint); ok {
			fileID = fid
		}

		return NewAliRAGModel(ctx, username, fileID)
	}

	// MCP 模型（集成MCP服务）
	f.creators["3"] = func(ctx context.Context, config map[string]interface{}) (AIModel, error) {
		username, ok := config["username"].(string)
		if !ok {
			return nil, fmt.Errorf("MCP model requires username")
		}
		return NewMCPModel(ctx, username)
	}

	//Ollama（目前提供接口实现，暂不提供应用，因为考虑到本地模型会占用很多空间）todo做
	f.creators["4"] = func(ctx context.Context, config map[string]interface{}) (AIModel, error) {
		baseURL, _ := config["baseURL"].(string)
		modelName, ok := config["modelName"].(string)
		if !ok {
			return nil, fmt.Errorf("Ollama model requires modelName")
		}
		return NewOllamaModel(ctx, baseURL, modelName)
	}
	// 阿里百炼 mcp 模型

}

// CreateAIModel 根据类型创建 AI 模型
func (f *AIModelFactory) CreateAIModel(ctx context.Context, modelType string, config map[string]interface{}) (AIModel, error) {
	creator, ok := f.creators[modelType]
	if !ok {
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
	return creator(ctx, config)
}

// CreateAIHelper 一键创建 AIHelper
func (f *AIModelFactory) CreateAIHelper(ctx context.Context, modelType string, SessionID string, config map[string]interface{}) (*AIHelper, error) {
	model, err := f.CreateAIModel(ctx, modelType, config)
	if err != nil {
		return nil, err
	}
	return NewAIHelper(model, SessionID), nil
}

// RegisterModel 可扩展注册
func (f *AIModelFactory) RegisterModel(modelType string, creator ModelCreator) {
	f.creators[modelType] = creator
}
