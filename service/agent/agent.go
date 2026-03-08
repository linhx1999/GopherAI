package agent

import (
	"context"
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

const (
	StreamPayloadTypeMeta  = "meta"
	StreamPayloadTypeError = "error"
)

type StreamMeta struct {
	Type         string `json:"type"`
	SessionID    string `json:"session_id"`
	MessageIndex int    `json:"message_index"`
}

type StreamError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// StreamEvent 表示 controller 可直接写入 SSE 的载荷。
type StreamEvent struct {
	Meta    *StreamMeta
	Message *schema.Message
	Error   *StreamError
}

// StreamHandle 表示一个可被 controller 消费的事件流。
type StreamHandle struct {
	SessionID string
	Events    <-chan StreamEvent
}

// AgentResult Agent 执行结果
type AgentResult struct {
	SessionID    string          `json:"session_id"`
	MessageIndex int             `json:"message_index"`
	Message      *schema.Message `json:"message"`
}

// HistoryMessageItem 历史消息项
type HistoryMessageItem struct {
	Index     int             `json:"index"`
	Message   *schema.Message `json:"message"`
	CreatedAt string          `json:"created_at"`
}

func NewErrorEvent(message string) StreamEvent {
	return StreamEvent{
		Error: &StreamError{
			Type:    StreamPayloadTypeError,
			Message: message,
		},
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

func appendMessageToRedis(msg *model.Message) error {
	return redis_cache.AppendMessage(msg.SessionID, msg)
}

func persistMessage(msg *model.Message) {
	if msg == nil {
		return
	}
	if err := appendMessageToRedis(msg); err != nil {
		log.Printf("appendMessageToRedis failed: %v", err)
	}
	publishMessageToDB(msg)
}

func buildStoredMessage(sessionID string, userName string, index int, msg *schema.Message) *model.Message {
	stored := &model.Message{
		SessionID: sessionID,
		UserName:  userName,
		Index:     index,
		CreatedAt: time.Now(),
	}
	if err := stored.SetSchemaMessage(msg); err != nil {
		log.Printf("SetSchemaMessage failed: %v", err)
	}
	return stored
}

func persistProducedMessages(sessionID string, userName string, startIndex int, produced []*schema.Message) int {
	lastIndex := startIndex - 1
	for i, msg := range produced {
		index := startIndex + i
		persistMessage(buildStoredMessage(sessionID, userName, index, msg))
		lastIndex = index
	}
	return lastIndex
}

func buildMessages(history []*model.Message, userMessage *schema.Message) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history)+2)
	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: SystemPrompt,
	})

	for _, m := range history {
		if m == nil {
			continue
		}
		msg := m.GetSchemaMessage()
		if msg != nil {
			messages = append(messages, msg)
		}
	}

	if userMessage != nil {
		messages = append(messages, userMessage)
	}

	return messages
}

func emitEvent(ctx context.Context, events chan<- StreamEvent, event StreamEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case events <- event:
		return nil
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

	userMessage := schema.UserMessage(userContent)
	nextIndex := len(history)
	persistMessage(buildStoredMessage(sessionID, userName, nextIndex, userMessage))

	input := buildMessages(history, userMessage)
	agentRunner, err := agentMgr.GetOrCreateAgentWithTools(ctx, sessionID, userName, toolNames, thinkingMode)
	if err != nil {
		log.Println("Generate GetOrCreateAgentWithTools error:", err)
		return nil, code.AIModelFail
	}

	produced, finalMessage, err := agentcommon.GenerateProducedMessages(ctx, agentRunner, input)
	if err != nil {
		log.Println("Generate error:", err)
		return nil, code.AIModelFail
	}
	if len(produced) == 0 && finalMessage != nil {
		produced = append(produced, finalMessage)
	}

	lastIndex := persistProducedMessages(sessionID, userName, nextIndex+1, produced)
	if finalMessage == nil && len(produced) > 0 {
		finalMessage = produced[len(produced)-1]
	}
	if finalMessage == nil {
		finalMessage = &schema.Message{Role: schema.Assistant}
	}
	if lastIndex < nextIndex+1 {
		lastIndex = nextIndex
	}

	return &AgentResult{
		SessionID:    sessionID,
		MessageIndex: lastIndex,
		Message:      finalMessage,
	}, code.CodeSuccess
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

	return openStreamWithSession(ctx, userName, sessionID, userContent, toolNames, thinkingMode, true)
}

func openStreamWithSession(ctx context.Context, userName string, sessionID string, userContent string, toolNames []string, thinkingMode bool, persistUser bool) (*StreamHandle, code.Code) {
	agentMgr := agentcommon.GetAgentManager()

	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("OpenStream getMessagesFromRedis error:", err)
		return nil, code.CodeServerBusy
	}

	var userMessage *schema.Message
	startIndex := len(history)
	if persistUser {
		userMessage = schema.UserMessage(userContent)
		persistMessage(buildStoredMessage(sessionID, userName, startIndex, userMessage))
		startIndex++
	}

	input := buildMessages(history, userMessage)
	agentRunner, err := agentMgr.GetOrCreateAgentWithTools(ctx, sessionID, userName, toolNames, thinkingMode)
	if err != nil {
		log.Println("OpenStream GetOrCreateAgentWithTools error:", err)
		return nil, code.AIModelFail
	}

	events := make(chan StreamEvent)
	go func() {
		defer close(events)

		if err := emitEvent(ctx, events, StreamEvent{
			Meta: &StreamMeta{
				Type:         StreamPayloadTypeMeta,
				SessionID:    sessionID,
				MessageIndex: startIndex,
			},
		}); err != nil {
			return
		}

		produced, err := agentcommon.StreamProducedMessages(ctx, agentRunner, input, func(msg *schema.Message) error {
			return emitEvent(ctx, events, StreamEvent{Message: msg})
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Println("OpenStream StreamProducedMessages error:", err)
			_ = emitEvent(ctx, events, NewErrorEvent("agent stream failed"))
			return
		}

		persistProducedMessages(sessionID, userName, startIndex, produced)
	}()

	return &StreamHandle{
		SessionID: sessionID,
		Events:    events,
	}, code.CodeSuccess
}

// GetMessages 获取会话消息列表
func GetMessages(sessionID string, userName string) ([]HistoryMessageItem, error) {
	msgs, err := message.GetMessagesBySessionIDOrdered(sessionID)
	if err != nil {
		return nil, err
	}

	items := make([]HistoryMessageItem, 0, len(msgs))
	for _, msg := range msgs {
		msgCopy := msg
		items = append(items, HistoryMessageItem{
			Index:     msg.Index,
			Message:   msgCopy.GetSchemaMessage(),
			CreatedAt: msg.CreatedAt.Format(time.RFC3339),
		})
	}
	return items, nil
}
