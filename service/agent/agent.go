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
	messageDAO "GopherAI/dao/message"
	sessionDAO "GopherAI/dao/session"
	"GopherAI/model"
)

const baseSystemPrompt = `你是一个智能助手，可以帮助用户解答各种问题。

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
	Meta  *StreamMeta
	Chunk *schema.Message
	Error *StreamError
}

// StreamHandle 表示一个可被 controller 消费的事件流。
type StreamHandle struct {
	SessionID string
	Events    <-chan StreamEvent
}

// GenerateRequest 表示一次消息生成请求。
// 已废弃：保留用于内部 prepareChatExecution，外部调用请使用具体参数。
type GenerateRequest struct {
	UserName         string
	SessionID        string
	UserMessage      string
	EnabledToolNames []string
	ThinkingMode     bool
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

type preparedChatExecution struct {
	sessionID                  string
	userName                   string
	conversation               []*schema.Message
	firstAssistantMessageIndex int
	agent                      *react.Agent
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

func appendMessageToCache(msg *model.Message) error {
	return messageDAO.AppendCachedMessage(msg.SessionID, msg)
}

func persistMessage(msg *model.Message) {
	if msg == nil {
		return
	}
	if err := appendMessageToCache(msg); err != nil {
		log.Printf("appendMessageToCache failed: %v", err)
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

func buildSystemPrompt(hasEnabledTools bool) string {
	if hasEnabledTools {
		return baseSystemPrompt + "\n\n当前对话已启用工具；当工具能提升答案质量时，请先调用合适的工具。若使用知识检索，请在回答中标注信息来源。"
	}
	return baseSystemPrompt + "\n\n当前对话未启用工具，请直接基于上下文作答。"
}

func buildConversationMessages(history []*model.Message, userMessage *schema.Message, hasEnabledTools bool) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history)+2)
	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: buildSystemPrompt(hasEnabledTools),
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

func checkOwnedSession(sessionID string, userName string) (*model.Session, code.Code) {
	session, err := sessionDAO.GetSessionByIDAndUserName(sessionID, userName)
	if err == nil {
		return session, code.CodeSuccess
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, code.CodeSessionNotFound
	}
	log.Printf("checkOwnedSession error: %v", err)
	return nil, code.CodeServerBusy
}

// loadSessionHistory 从 Redis 获取消息历史，并在缓存未命中时回源数据库。
func loadSessionHistory(sessionID string, userName string) ([]*model.Message, error) {
	cachedMessages, err := messageDAO.GetCachedMessages(sessionID)
	if err != nil {
		log.Printf("loadSessionHistory redis error: %v", err)
	} else if len(cachedMessages) > 0 {
		return cachedMessages, nil
	}

	dbMessages, err := messageDAO.ListMessagesBySessionAndUserOrdered(sessionID, userName)
	if err != nil {
		log.Printf("loadSessionHistory db error: %v", err)
		return nil, err
	}

	history := make([]*model.Message, 0, len(dbMessages))
	for i := range dbMessages {
		history = append(history, &dbMessages[i])
	}

	if len(history) > 0 {
		go func() {
			if err := messageDAO.StoreCachedMessages(sessionID, history); err != nil {
				log.Printf("loadSessionHistory cache to redis error: %v", err)
			}
		}()
	}

	return history, nil
}

func buildHistoryMessageItems(msgs []*model.Message) []HistoryMessageItem {
	items := make([]HistoryMessageItem, 0, len(msgs))
	for _, msg := range msgs {
		msgCopy := msg
		items = append(items, HistoryMessageItem{
			Index:     msg.Index,
			Message:   msgCopy.GetSchemaMessage(),
			CreatedAt: msg.CreatedAt.Format(time.RFC3339),
		})
	}
	return items
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

func prepareChatExecution(ctx context.Context, req GenerateRequest) (*preparedChatExecution, code.Code) {
	sessionID := req.SessionID
	if sessionID == "" {
		var code_ code.Code
		sessionID, code_ = createSession(req.UserName, req.UserMessage)
		if code_ != code.CodeSuccess {
			return nil, code_
		}
	} else {
		if _, code_ := checkOwnedSession(sessionID, req.UserName); code_ != code.CodeSuccess {
			return nil, code_
		}
	}

	history, err := loadSessionHistory(sessionID, req.UserName)
	if err != nil {
		log.Println("prepareChatExecution loadSessionHistory error:", err)
		return nil, code.CodeServerBusy
	}

	userMessageIndex := len(history)
	userMessage := schema.UserMessage(req.UserMessage)
	persistMessage(buildStoredMessage(sessionID, req.UserName, userMessageIndex, userMessage))

	agent, err := agentcommon.GetAgentManager().GetOrCreateAgentForChat(ctx, sessionID, req.UserName, req.EnabledToolNames, req.ThinkingMode)
	if err != nil {
		log.Println("prepareChatExecution GetOrCreateAgentForChat error:", err)
		return nil, code.AIModelFail
	}

	hasEnabledTools := len(req.EnabledToolNames) > 0

	return &preparedChatExecution{
		sessionID:                  sessionID,
		userName:                   req.UserName,
		conversation:               buildConversationMessages(history, userMessage, hasEnabledTools),
		firstAssistantMessageIndex: userMessageIndex + 1,
		agent:                      agent,
	}, code.CodeSuccess
}

// Generate 同步生成响应。
func Generate(ctx context.Context, userName, sessionID, userMessage string, enabledToolNames []string, thinkingMode bool) (*GenerateResult, code.Code) {
	run, code_ := prepareChatExecution(ctx, GenerateRequest{
		UserName:         userName,
		SessionID:        sessionID,
		UserMessage:      userMessage,
		EnabledToolNames: enabledToolNames,
		ThinkingMode:     thinkingMode,
	})
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	produced, finalMessage, err := agentcommon.CollectAgentMessages(ctx, run.agent, run.conversation)
	if err != nil {
		log.Println("Generate error:", err)
		return nil, code.AIModelFail
	}
	if len(produced) == 0 && finalMessage != nil {
		produced = append(produced, finalMessage)
	}

	lastIndex := persistProducedMessages(run.sessionID, run.userName, run.firstAssistantMessageIndex, produced)
	if finalMessage == nil && len(produced) > 0 {
		finalMessage = produced[len(produced)-1]
	}
	if finalMessage == nil {
		finalMessage = &schema.Message{Role: schema.Assistant}
	}
	if lastIndex < run.firstAssistantMessageIndex {
		lastIndex = run.firstAssistantMessageIndex - 1
	}

	return &GenerateResult{
		SessionID:    run.sessionID,
		MessageIndex: lastIndex,
		Message:      finalMessage,
	}, code.CodeSuccess
}

// Stream 打开一个可供 controller 消费的 SSE 事件流。
func Stream(ctx context.Context, userName, sessionID, userMessage string, enabledToolNames []string, thinkingMode bool) (*StreamHandle, code.Code) {
	run, code_ := prepareChatExecution(ctx, GenerateRequest{
		UserName:         userName,
		SessionID:        sessionID,
		UserMessage:      userMessage,
		EnabledToolNames: enabledToolNames,
		ThinkingMode:     thinkingMode,
	})
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
				MessageIndex: run.firstAssistantMessageIndex,
			},
		}); err != nil {
			return
		}

		produced, err := agentcommon.StreamAgentMessages(ctx, run.agent, run.conversation, func(msg *schema.Message) error {
			return emitEvent(ctx, events, StreamEvent{Chunk: msg})
		})
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Println("Stream StreamAgentMessages error:", err)
			_ = emitEvent(ctx, events, NewErrorEvent("agent stream failed"))
			return
		}

		persistProducedMessages(run.sessionID, run.userName, run.firstAssistantMessageIndex, produced)
	}()

	return &StreamHandle{
		SessionID: run.sessionID,
		Events:    events,
	}, code.CodeSuccess
}

// ListHistoryMessages 获取当前用户的会话消息列表。
func ListHistoryMessages(sessionID string, userName string) ([]HistoryMessageItem, code.Code) {
	if _, code_ := checkOwnedSession(sessionID, userName); code_ != code.CodeSuccess {
		return nil, code_
	}

	msgs, err := loadSessionHistory(sessionID, userName)
	if err != nil {
		log.Println("ListHistoryMessages error:", err)
		return nil, code.CodeServerBusy
	}

	return buildHistoryMessageItems(msgs), code.CodeSuccess
}
