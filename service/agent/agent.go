package agent

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	agentcommon "GopherAI/common/agent"
	"GopherAI/common/code"
	"GopherAI/common/rabbitmq"
	redis_cache "GopherAI/common/redis"
	"GopherAI/dao/message"
	"GopherAI/dao/session"
	"GopherAI/model"
)

var ctx = context.Background()

// SystemPrompt Agent 系统提示词
const SystemPrompt = `你是一个智能助手，可以帮助用户解答各种问题。

当你需要查询知识库中的信息时，请使用 knowledge_search 工具进行检索。
当需要逐步分析复杂问题时，请使用 sequential_thinking 工具。
请在回答时标注信息来源（如果使用了知识库检索）。

请用中文回答，保持回答简洁、准确、友好。`

// SSE 事件类型
const (
	SSEEventTypeMeta         = "meta"
	SSEEventTypeToolCall     = "tool_call"
	SSEEventTypeReasoning    = "reasoning_delta"
	SSEEventTypeReasoningEnd = "reasoning_end"
	SSEEventTypeContentDelta = "content_delta"
	SSEEventTypeMessageEnd   = "message_end"
	SSEEventTypeError        = "error"
)

// SSEEvent 统一的 SSE 载荷结构。
type SSEEvent struct {
	Type         string          `json:"type"`
	SessionID    string          `json:"session_id,omitempty"`
	MessageIndex int             `json:"message_index,omitempty"`
	ToolID       string          `json:"tool_id,omitempty"`
	Function     string          `json:"function,omitempty"`
	Arguments    json.RawMessage `json:"arguments,omitempty"`
	Content      string          `json:"content,omitempty"`
	Status       string          `json:"status,omitempty"`
	Message      string          `json:"message,omitempty"`
}

// StreamHandle 表示一个可被 controller 消费的事件流。
type StreamHandle struct {
	SessionID string
	Events    <-chan SSEEvent
}

// AgentResult Agent 执行结果
type AgentResult struct {
	SessionID        string           `json:"session_id"`
	MessageIndex     int              `json:"message_index"`
	Role             string           `json:"role"`
	Content          string           `json:"content"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	ToolCalls        []model.ToolCall `json:"tool_calls,omitempty"`
}

// NewErrorEvent 创建错误事件。
func NewErrorEvent(message string) SSEEvent {
	return SSEEvent{
		Type:    SSEEventTypeError,
		Message: message,
	}
}

// publishMessageToDB 异步保存消息到数据库（通过 RabbitMQ）。
func publishMessageToDB(msg *model.Message) {
	data, err := rabbitmq.GenerateMessageMQParam(msg)
	if err != nil {
		log.Printf("publishMessageToDB marshal error: %v", err)
		return
	}
	if err := rabbitmq.RMQMessage.Publish(data); err != nil {
		log.Printf("publishMessageToDB publish error: %v", err)
	}
}

// appendMessageToRedis 追加消息到 Redis 缓存
func appendMessageToRedis(msg *model.Message) error {
	return redis_cache.AppendMessage(msg.SessionID, msg)
}

func buildUserMessage(sessionID string, userName string, index int, content string) *model.Message {
	return &model.Message{
		SessionID: sessionID,
		UserName:  userName,
		Index:     index,
		Role:      "user",
		Content:   content,
		CreatedAt: time.Now(),
	}
}

func buildAssistantMessage(sessionID string, userName string, index int, result *agentcommon.StreamResult) *model.Message {
	msg := &model.Message{
		SessionID: sessionID,
		UserName:  userName,
		Index:     index,
		Role:      "assistant",
		Content:   result.Content,
		CreatedAt: time.Now(),
	}
	if len(result.ToolCalls) > 0 {
		if err := msg.SetToolCalls(result.ToolCalls); err != nil {
			log.Printf("buildAssistantMessage set tool calls failed: %v", err)
		}
	}
	return msg
}

func buildAgentResult(sessionID string, messageIndex int, result *agentcommon.StreamResult) *AgentResult {
	return &AgentResult{
		SessionID:        sessionID,
		MessageIndex:     messageIndex,
		Role:             "assistant",
		Content:          result.Content,
		ReasoningContent: result.ReasoningContent,
		ToolCalls:        result.ToolCalls,
	}
}

func emitEvent(ctx context.Context, events chan<- SSEEvent, event SSEEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case events <- event:
		return nil
	}
}

func translateStreamEvent(event agentcommon.StreamEvent) SSEEvent {
	switch event.Type {
	case agentcommon.StreamEventTypeToolCall:
		if event.ToolCall == nil {
			return SSEEvent{}
		}
		return SSEEvent{
			Type:      SSEEventTypeToolCall,
			ToolID:    event.ToolCall.ToolID,
			Function:  event.ToolCall.Function,
			Arguments: event.ToolCall.Arguments,
		}
	case agentcommon.StreamEventTypeReasoningDelta:
		return SSEEvent{
			Type:    SSEEventTypeReasoning,
			Content: event.Content,
		}
	case agentcommon.StreamEventTypeReasoningEnd:
		return SSEEvent{
			Type:   SSEEventTypeReasoningEnd,
			Status: "completed",
		}
	case agentcommon.StreamEventTypeContentDelta:
		return SSEEvent{
			Type:    SSEEventTypeContentDelta,
			Content: event.Content,
		}
	default:
		return SSEEvent{}
	}
}

// getMessagesFromRedis 从 Redis 获取消息历史（Cache-Aside 模式）
func getMessagesFromRedis(sessionID string) ([]*model.Message, error) {
	messages, err := redis_cache.GetMessages(sessionID)
	if err != nil {
		log.Printf("getMessagesFromRedis redis error: %v", err)
	} else if len(messages) > 0 {
		return messages, nil
	}

	dbMessages, err := message.GetMessagesBySessionIDOrdered(sessionID)
	if err != nil {
		log.Printf("getMessagesFromRedis db error: %v", err)
		return nil, err
	}

	result := make([]*model.Message, 0, len(dbMessages))
	for i := range dbMessages {
		result = append(result, &dbMessages[i])
	}

	if len(result) > 0 {
		go func() {
			if err := redis_cache.CacheMessages(sessionID, result); err != nil {
				log.Printf("getMessagesFromRedis cache to redis error: %v", err)
			}
		}()
	}

	return result, nil
}

// buildMessages 构建发送给 Agent 的消息列表
func buildMessages(history []*model.Message, userContent string) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history)+2)
	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: SystemPrompt,
	})

	for _, m := range history {
		role := schema.Assistant
		if m.Role == "user" {
			role = schema.User
		}
		messages = append(messages, &schema.Message{
			Role:    role,
			Content: m.Content,
		})
	}

	if userContent != "" {
		messages = append(messages, &schema.Message{
			Role:    schema.User,
			Content: userContent,
		})
	}

	return messages
}

// CreateSessionOnly 仅创建会话（用于流式响应）
func CreateSessionOnly(userName string, title string) (string, code.Code) {
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    title,
	}
	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateSessionOnly error:", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

// Generate 同步生成响应
func Generate(userName string, sessionID string, userContent string, toolNames []string, thinkingMode bool) (*AgentResult, code.Code) {
	return GenerateWithContext(ctx, userName, sessionID, userContent, toolNames, thinkingMode)
}

// GenerateWithContext 带上下文的同步生成响应
func GenerateWithContext(ctx context.Context, userName string, sessionID string, userContent string, toolNames []string, thinkingMode bool) (*AgentResult, code.Code) {
	agentMgr := agentcommon.GetAgentManager()

	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("Generate getMessagesFromRedis error:", err)
		return nil, code.CodeServerBusy
	}

	nextIndex := len(history)

	userMsg := buildUserMessage(sessionID, userName, nextIndex, userContent)
	_ = appendMessageToRedis(userMsg)
	publishMessageToDB(userMsg)

	messages := buildMessages(history, userContent)

	aiResponse, err := agentMgr.GenerateWithTools(ctx, sessionID, userName, messages, toolNames, thinkingMode)
	if err != nil {
		log.Println("Generate error:", err)
		return nil, code.AIModelFail
	}

	result := agentcommon.ResultFromMessage(aiResponse)
	aiMsg := buildAssistantMessage(sessionID, userName, nextIndex+1, result)
	_ = appendMessageToRedis(aiMsg)
	publishMessageToDB(aiMsg)

	return buildAgentResult(sessionID, aiMsg.Index, result), code.CodeSuccess
}

// OpenStreamWithMeta 打开一个可供 controller 消费的 SSE 事件流。
func OpenStreamWithMeta(ctx context.Context, userName string, sessionID string, userContent string, toolNames []string, thinkingMode bool) (*StreamHandle, code.Code) {
	if sessionID == "" {
		var code_ code.Code
		sessionID, code_ = CreateSessionOnly(userName, userContent)
		if code_ != code.CodeSuccess {
			return nil, code_
		}
	}

	return openStreamWithSession(ctx, userName, sessionID, userContent, toolNames, thinkingMode)
}

func openStreamWithSession(ctx context.Context, userName string, sessionID string, userContent string, toolNames []string, thinkingMode bool) (*StreamHandle, code.Code) {
	agentMgr := agentcommon.GetAgentManager()

	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("OpenStream getMessagesFromRedis error:", err)
		return nil, code.CodeServerBusy
	}

	nextIndex := len(history)
	aiIndex := nextIndex + 1

	userMsg := buildUserMessage(sessionID, userName, nextIndex, userContent)
	_ = appendMessageToRedis(userMsg)
	publishMessageToDB(userMsg)

	messages := buildMessages(history, userContent)
	msgReader, err := agentMgr.OpenStreamWithTools(ctx, sessionID, userName, messages, toolNames, thinkingMode)
	if err != nil {
		log.Println("OpenStream OpenStreamWithTools error:", err)
		return nil, code.AIModelFail
	}

	events := make(chan SSEEvent)
	go func() {
		defer close(events)

		if err := emitEvent(ctx, events, SSEEvent{
			Type:         SSEEventTypeMeta,
			SessionID:    sessionID,
			MessageIndex: aiIndex,
		}); err != nil {
			return
		}

		result, err := agentcommon.ConsumeStream(msgReader, thinkingMode, func(event agentcommon.StreamEvent) error {
			sseEvent := translateStreamEvent(event)
			if sseEvent.Type == "" {
				return nil
			}
			return emitEvent(ctx, events, sseEvent)
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Println("OpenStream consume error:", err)
			_ = emitEvent(ctx, events, SSEEvent{
				Type:   SSEEventTypeMessageEnd,
				Status: "error",
			})
			return
		}

		if err := emitEvent(ctx, events, SSEEvent{
			Type:   SSEEventTypeMessageEnd,
			Status: "completed",
		}); err != nil {
			return
		}

		aiMsg := buildAssistantMessage(sessionID, userName, aiIndex, result)
		_ = appendMessageToRedis(aiMsg)
		publishMessageToDB(aiMsg)
	}()

	return &StreamHandle{
		SessionID: sessionID,
		Events:    events,
	}, code.CodeSuccess
}

// OpenRegenerateStream 打开重新生成的 SSE 事件流。
func OpenRegenerateStream(ctx context.Context, userName string, sessionID string, fromIndex int, toolNames []string, thinkingMode bool) (*StreamHandle, code.Code) {
	if code_ := validateRegenerateSession(sessionID, fromIndex); code_ != code.CodeSuccess {
		return nil, code_
	}

	redis_cache.DeleteMessages(sessionID)
	agentcommon.GetAgentManager().ClearAgent(sessionID)

	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		return nil, code.CodeServerBusy
	}

	lastUserContent := getLastUserContent(history)
	if lastUserContent == "" {
		return nil, code.CodeInvalidParams
	}

	return openStreamWithSession(ctx, userName, sessionID, lastUserContent, toolNames, thinkingMode)
}

// Regenerate 重新生成同步响应
func Regenerate(ctx context.Context, userName string, sessionID string, fromIndex int, toolNames []string, thinkingMode bool) (*AgentResult, code.Code) {
	if code_ := validateRegenerateSession(sessionID, fromIndex); code_ != code.CodeSuccess {
		return nil, code_
	}

	redis_cache.DeleteMessages(sessionID)
	agentcommon.GetAgentManager().ClearAgent(sessionID)

	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		return nil, code.CodeServerBusy
	}

	lastUserContent := getLastUserContent(history)
	if lastUserContent == "" {
		return nil, code.CodeInvalidParams
	}

	return GenerateWithContext(ctx, userName, sessionID, lastUserContent, toolNames, thinkingMode)
}

func validateRegenerateSession(sessionID string, fromIndex int) code.Code {
	sess, err := session.GetSessionByID(sessionID)
	if err != nil || sess == nil {
		return code.CodeSessionNotFound
	}

	count, err := message.GetMessageCount(sessionID)
	if err != nil {
		return code.CodeServerBusy
	}
	if fromIndex < 0 || fromIndex >= count {
		return code.CodeInvalidParams
	}

	if err := message.TruncateMessages(sessionID, fromIndex); err != nil {
		return code.CodeServerBusy
	}

	return code.CodeSuccess
}

func getLastUserContent(history []*model.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			return history[i].Content
		}
	}
	return ""
}

// GetMessages 获取会话消息列表
func GetMessages(sessionID string, userName string) ([]model.Message, error) {
	return message.GetMessagesBySessionIDOrdered(sessionID)
}
