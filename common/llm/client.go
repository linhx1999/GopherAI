package llm

import (
	"GopherAI/config"
	"GopherAI/model"
	"GopherAI/utils"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
	eino_model "github.com/cloudwego/eino/components/model"
)

// StreamCallback 流式响应回调函数
type StreamCallback func(msg string)

// LLMClient 统一的大模型客户端
type LLMClient struct {
	llm eino_model.ToolCallingChatModel
	mu  sync.RWMutex
}

var (
	globalClient *LLMClient
	clientOnce   sync.Once
)

// GetLLMClient 获取全局 LLM 客户端实例
func GetLLMClient() *LLMClient {
	clientOnce.Do(func() {
		globalClient = &LLMClient{}
	})
	return globalClient
}

// Init 初始化 LLM 客户端
func (c *LLMClient) Init(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conf := config.GetConfig()
	apiKey := conf.OpenAIConfig.ApiKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}

	modelName := conf.OpenAIConfig.ModelName
	if modelName == "" {
		modelName = os.Getenv("OPENAI_MODEL_NAME")
	}

	baseURL := conf.OpenAIConfig.BaseUrl
	if baseURL == "" {
		baseURL = os.Getenv("OPENAI_BASE_URL")
	}

	llm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
		APIKey:  apiKey,
	})
	if err != nil {
		return fmt.Errorf("create openai model failed: %v", err)
	}

	c.llm = llm
	return nil
}

// Generate 同步生成响应
func (c *LLMClient) Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.llm == nil {
		return nil, fmt.Errorf("llm client not initialized")
	}

	resp, err := c.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("llm generate failed: %v", err)
	}

	return resp, nil
}

// Stream 流式生成响应
func (c *LLMClient) Stream(ctx context.Context, messages []*schema.Message, cb StreamCallback) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.llm == nil {
		return "", fmt.Errorf("llm client not initialized")
	}

	stream, err := c.llm.Stream(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("llm stream failed: %v", err)
	}
	defer stream.Close()

	var fullResp strings.Builder

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("llm stream recv failed: %v", err)
		}
		if len(msg.Content) > 0 {
			fullResp.WriteString(msg.Content)
			if cb != nil {
				cb(msg.Content)
			}
		}
	}

	return fullResp.String(), nil
}

// ConvertMessages 将 model.Message 转换为 schema.Message
func ConvertMessages(messages []*model.Message) []*schema.Message {
	return utils.ConvertToSchemaMessages(messages)
}
