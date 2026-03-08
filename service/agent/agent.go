package agent

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"gorm.io/gorm"

	agentcommon "GopherAI/common/agent"
	"GopherAI/common/code"
	"GopherAI/common/rabbitmq"
	redis_cache "GopherAI/common/redis"
	messageDAO "GopherAI/dao/message"
	sessionDAO "GopherAI/dao/session"
	"GopherAI/model"
)

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

// GenerateRequest 表示一次消息生成请求。
type GenerateRequest struct {
	UserName     string
	SessionID    string
	UserMessage  string
	ToolNames    []string
	ThinkingMode bool
}

// GenerateResult 表示非流式生成结果。
type GenerateResult struct {
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

type preparedAgentRun struct {
	sessionID        string
	userName         string
	prompt           []*schema.Message
	outputStartIndex int
	runner           *react.Agent
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

func loadOwnedSession(sessionID string, userName string) (*model.Session, code.Code) {
	session, err := sessionDAO.GetSessionByIDAndUserName(sessionID, userName)
	if err == nil {
		return session, code.CodeSuccess
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, code.CodeSessionNotFound
	}
	log.Printf("loadOwnedSession error: %v", err)
	return nil, code.CodeServerBusy
}

// getSessionHistory 从 Redis 获取消息历史，并在缓存未命中时回源数据库。
func getSessionHistory(sessionID string, userName string) ([]*model.Message, error) {
	cachedMessages, err := redis_cache.GetMessages(sessionID)
	if err != nil {
		log.Printf("getSessionHistory redis error: %v", err)
	} else if len(cachedMessages) > 0 {
		return cachedMessages, nil
	}

	dbMessages, err := messageDAO.ListMessagesBySessionAndUserOrdered(sessionID, userName)
	if err != nil {
		log.Printf("getSessionHistory db error: %v", err)
		return nil, err
	}

	history := make([]*model.Message, 0, len(dbMessages))
	for i := range dbMessages {
		history = append(history, &dbMessages[i])
	}

	if len(history) > 0 {
		go func() {
			if err := redis_cache.CacheMessages(sessionID, history); err != nil {
				log.Printf("getSessionHistory cache to redis error: %v", err)
			}
		}()
	}

	return history, nil
}

func createSession(userName string, title string) (string, code.Code) {
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    title,
	}
	createdSession, err := sessionDAO.CreateSession(newSession)
	if err != nil {
		log.Println("createSession error:", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

func prepareAgentRun(ctx context.Context, req GenerateRequest) (*preparedAgentRun, code.Code) {
	sessionID := req.SessionID
	if sessionID == "" {
		var code_ code.Code
		sessionID, code_ = createSession(req.UserName, req.UserMessage)
		if code_ != code.CodeSuccess {
			return nil, code_
		}
	} else {
		if _, code_ := loadOwnedSession(sessionID, req.UserName); code_ != code.CodeSuccess {
			return nil, code_
		}
	}

	history, err := getSessionHistory(sessionID, req.UserName)
	if err != nil {
		log.Println("prepareAgentRun getSessionHistory error:", err)
		return nil, code.CodeServerBusy
	}

	userMessageIndex := len(history)
	userMessage := schema.UserMessage(req.UserMessage)
	persistMessage(buildStoredMessage(sessionID, req.UserName, userMessageIndex, userMessage))

	runner, err := agentcommon.GetAgentManager().GetOrCreateAgentWithTools(ctx, sessionID, req.UserName, req.ToolNames, req.ThinkingMode)
	if err != nil {
		log.Println("prepareAgentRun GetOrCreateAgentWithTools error:", err)
		return nil, code.AIModelFail
	}

	return &preparedAgentRun{
		sessionID:        sessionID,
		userName:         req.UserName,
		prompt:           buildMessages(history, userMessage),
		outputStartIndex: userMessageIndex + 1,
		runner:           runner,
	}, code.CodeSuccess
}

// Generate 同步生成响应。
func Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, code.Code) {
	run, code_ := prepareAgentRun(ctx, req)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	produced, finalMessage, err := agentcommon.GenerateProducedMessages(ctx, run.runner, run.prompt)
	if err != nil {
		log.Println("Generate error:", err)
		return nil, code.AIModelFail
	}
	if len(produced) == 0 && finalMessage != nil {
		produced = append(produced, finalMessage)
	}

	lastIndex := persistProducedMessages(run.sessionID, run.userName, run.outputStartIndex, produced)
	if finalMessage == nil && len(produced) > 0 {
		finalMessage = produced[len(produced)-1]
	}
	if finalMessage == nil {
		finalMessage = &schema.Message{Role: schema.Assistant}
	}
	if lastIndex < run.outputStartIndex {
		lastIndex = run.outputStartIndex - 1
	}

	return &GenerateResult{
		SessionID:    run.sessionID,
		MessageIndex: lastIndex,
		Message:      finalMessage,
	}, code.CodeSuccess
}

// Stream 打开一个可供 controller 消费的 SSE 事件流。
func Stream(ctx context.Context, req GenerateRequest) (*StreamHandle, code.Code) {
	run, code_ := prepareAgentRun(ctx, req)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	events := make(chan StreamEvent)
	go func() {
		defer close(events)

		if err := emitEvent(ctx, events, StreamEvent{
			Meta: &StreamMeta{
				Type:         StreamPayloadTypeMeta,
				SessionID:    run.sessionID,
				MessageIndex: run.outputStartIndex,
			},
		}); err != nil {
			return
		}

		produced, err := agentcommon.StreamProducedMessages(ctx, run.runner, run.prompt, func(msg *schema.Message) error {
			return emitEvent(ctx, events, StreamEvent{Message: msg})
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Println("Stream StreamProducedMessages error:", err)
			_ = emitEvent(ctx, events, NewErrorEvent("agent stream failed"))
			return
		}

		persistProducedMessages(run.sessionID, run.userName, run.outputStartIndex, produced)
	}()

	return &StreamHandle{
		SessionID: run.sessionID,
		Events:    events,
	}, code.CodeSuccess
}

// ListHistoryMessages 获取当前用户的会话消息列表。
func ListHistoryMessages(sessionID string, userName string) ([]HistoryMessageItem, code.Code) {
	if _, code_ := loadOwnedSession(sessionID, userName); code_ != code.CodeSuccess {
		return nil, code_
	}

	msgs, err := messageDAO.ListMessagesBySessionAndUserOrdered(sessionID, userName)
	if err != nil {
		log.Println("ListHistoryMessages error:", err)
		return nil, code.CodeServerBusy
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
	return items, code.CodeSuccess
}
