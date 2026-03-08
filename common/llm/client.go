package llm

import (
	"GopherAI/config"
	"GopherAI/model"
	"GopherAI/utils"
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"
	eino_model "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// LLMClient 统一的大模型客户端
type LLMClient struct {
	models map[string]eino_model.ToolCallingChatModel
	mu     sync.RWMutex
}

var (
	globalClient *LLMClient
	clientOnce   sync.Once
)

// GetLLMClient 获取全局 LLM 客户端实例
func GetLLMClient() *LLMClient {
	clientOnce.Do(func() {
		globalClient = &LLMClient{
			models: make(map[string]eino_model.ToolCallingChatModel),
		}
	})
	return globalClient
}

// Init 初始化 LLM 客户端
func (c *LLMClient) Init(ctx context.Context) error {
	_, _, err := c.GetModelForMode(ctx, false)
	return err
}

// Generate 同步生成响应
func (c *LLMClient) Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	model, _, err := c.GetModelForMode(ctx, false)
	if err != nil {
		return nil, err
	}

	resp, err := model.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("llm generate failed: %v", err)
	}

	return resp, nil
}

func getOpenAIAPIKey() string {
	conf := config.GetConfig()
	if conf.OpenAIConfig.ApiKey != "" {
		return conf.OpenAIConfig.ApiKey
	}
	return os.Getenv("OPENAI_API_KEY")
}

func getOpenAIBaseURL() string {
	conf := config.GetConfig()
	if conf.OpenAIConfig.BaseUrl != "" {
		return conf.OpenAIConfig.BaseUrl
	}
	return os.Getenv("OPENAI_BASE_URL")
}

func getDefaultModelName() string {
	conf := config.GetConfig()
	if conf.OpenAIConfig.ModelName != "" {
		return conf.OpenAIConfig.ModelName
	}
	return os.Getenv("OPENAI_MODEL_NAME")
}

func getReasoningModelName() string {
	conf := config.GetConfig()
	if conf.OpenAIConfig.ReasoningModelName != "" {
		return conf.OpenAIConfig.ReasoningModelName
	}
	if modelName := os.Getenv("OPENAI_REASONING_MODEL_NAME"); modelName != "" {
		return modelName
	}
	return getDefaultModelName()
}

func (c *LLMClient) getOrCreateModel(ctx context.Context, modelName string) (eino_model.ToolCallingChatModel, error) {
	if modelName == "" {
		return nil, fmt.Errorf("model name is empty")
	}

	c.mu.RLock()
	if cached, ok := c.models[modelName]; ok {
		c.mu.RUnlock()
		return cached, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, ok := c.models[modelName]; ok {
		return cached, nil
	}

	apiKey := getOpenAIAPIKey()
	if apiKey == "" {
		return nil, fmt.Errorf("openai api key is empty")
	}

	llm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: getOpenAIBaseURL(),
		Model:   modelName,
		APIKey:  apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("create openai model failed: %v", err)
	}

	c.models[modelName] = llm
	return llm, nil
}

// GetModelForMode 按模式获取底层模型。
func (c *LLMClient) GetModelForMode(ctx context.Context, thinkingMode bool) (eino_model.ToolCallingChatModel, string, error) {
	modelName := getDefaultModelName()
	if thinkingMode {
		modelName = getReasoningModelName()
	}

	model, err := c.getOrCreateModel(ctx, modelName)
	if err != nil {
		return nil, "", err
	}
	return model, modelName, nil
}

// ConvertMessages 将 model.Message 转换为 schema.Message
func ConvertMessages(messages []*model.Message) []*schema.Message {
	return utils.ConvertToSchemaMessages(messages)
}
