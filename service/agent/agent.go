package agent

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"gorm.io/gorm"

	agentcommon "GopherAI/common/agent"
	agenttools "GopherAI/common/agent/tools"
	"GopherAI/common/code"
	"GopherAI/common/rabbitmq"
	messageDAO "GopherAI/dao/message"
	sessionDAO "GopherAI/dao/session"
	"GopherAI/model"
)

const baseSystemPrompt = `你是一个智能助手，可以帮助用户解答各种问题。

请用中文回答，保持回答简洁、准确、友好。`

const (
	StreamEventTypeResponseCreated          = "response.created"
	StreamEventTypeResponseMessageDelta     = "response.message.delta"
	StreamEventTypeResponseMessageCompleted = "response.message.completed"
	StreamEventTypeResponseError            = "response.error"
	StreamEventTypeResponseDone             = "response.done"
)

type StreamEnvelope struct {
	Type     string      `json:"type"`
	Code     code.Code   `json:"code"`
	Message  string      `json:"message"`
	Response interface{} `json:"response"`
}

// StreamHandle 表示一个可被 controller 消费的事件流。
type StreamHandle struct {
	SessionID string
	Events    <-chan StreamEnvelope
}

// GenerateResult 表示非流式生成结果。
type GenerateResult struct {
	SessionID    string          `json:"session_id"`
	MessageIndex int             `json:"message_index"`
	Message      *schema.Message `json:"message"`
}

// HistoryMessageItem 历史消息项
type HistoryMessageItem struct {
	MessageID string          `json:"message_id"`
	Index     int             `json:"index"`
	Message   *schema.Message `json:"message"`
	CreatedAt string          `json:"created_at"`
}

type preparedChatExecution struct {
	session                    *model.Session
	userRefID                  uint
	conversation               []*schema.Message
	firstAssistantMessageIndex int
	agent                      adk.Agent
}

type StreamDeltaResponse struct {
	Delta *schema.Message `json:"delta"`
}

type StreamCompletedResponse struct {
	Message *schema.Message `json:"message"`
}

func NewSuccessEvent(eventType string, response interface{}) StreamEnvelope {
	return StreamEnvelope{
		Type:     eventType,
		Code:     code.CodeSuccess,
		Message:  code.CodeSuccess.Msg(),
		Response: response,
	}
}

func NewErrorEvent(code_ code.Code, message string) StreamEnvelope {
	errorMsg := strings.TrimSpace(message)
	if errorMsg == "" {
		errorMsg = code_.Msg()
	}
	return StreamEnvelope{
		Type:     StreamEventTypeResponseError,
		Code:     code_,
		Message:  errorMsg,
		Response: nil,
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
	return nil
}

func persistMessage(sessionID string, msg *model.Message) {
	if msg == nil {
		return
	}
	if err := messageDAO.AppendCachedMessage(sessionID, msg); err != nil {
		log.Printf("appendMessageToCache failed: %v", err)
	}
	publishMessageToDB(msg)
}

func buildStoredMessage(session *model.Session, userRefID uint, index int, msg *schema.Message) *model.Message {
	stored := &model.Message{
		MessageID:    uuid.New().String(),
		SessionRefID: session.ID,
		UserRefID:    userRefID,
		Index:        index,
	}
	stored.CreatedAt = time.Now()
	if err := stored.SetSchemaMessage(msg); err != nil {
		log.Printf("SetSchemaMessage failed: %v", err)
	}
	return stored
}

func persistProducedMessages(session *model.Session, userRefID uint, startIndex int, produced []*schema.Message) int {
	lastIndex := startIndex - 1
	for i, msg := range produced {
		index := startIndex + i
		persistMessage(session.SessionID, buildStoredMessage(session, userRefID, index, msg))
		lastIndex = index
	}
	return lastIndex
}

func buildAgentInstruction(hasEnabledTools bool) string {
	if hasEnabledTools {
		return baseSystemPrompt + "\n\n当前对话已启用工具；当工具能提升答案质量时，请先调用合适的工具。若使用知识检索，请在回答中标注信息来源。"
	}
	return baseSystemPrompt + "\n\n当前对话未启用工具，请直接基于上下文作答。"
}

func buildConversationMessages(history []*model.Message, userMessage *schema.Message) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history)+1)
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

func emitEvent(ctx context.Context, events chan<- StreamEnvelope, event StreamEnvelope) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case events <- event:
		return nil
	}
}

func isRequestCanceled(ctx context.Context, err error) bool {
	if ctx != nil {
		if ctxErr := ctx.Err(); errors.Is(ctxErr, context.Canceled) || errors.Is(ctxErr, context.DeadlineExceeded) {
			return true
		}
	}
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "context canceled") || strings.Contains(errText, "context deadline exceeded")
}

func checkOwnedSession(sessionID string, userRefID uint) (*model.Session, code.Code) {
	session, err := sessionDAO.GetSessionBySessionIDAndUserRefID(sessionID, userRefID)
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
func loadSessionHistory(session *model.Session, userRefID uint) ([]*model.Message, error) {
	cachedMessages, err := messageDAO.GetCachedMessages(session.SessionID)
	if err != nil {
		log.Printf("loadSessionHistory redis error: %v", err)
	} else if len(cachedMessages) > 0 {
		return cachedMessages, nil
	}

	dbMessages, err := messageDAO.ListMessagesBySessionRefIDAndUserRefIDOrdered(session.ID, userRefID)
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
			if err := messageDAO.StoreCachedMessages(session.SessionID, history); err != nil {
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
			MessageID: msg.MessageID,
			Index:     msg.Index,
			Message:   msgCopy.GetSchemaMessage(),
			CreatedAt: msg.CreatedAt.Format(time.RFC3339),
		})
	}
	return items
}

func syncSessionTitleWithFirstMessage(session *model.Session, userRefID uint, history []*model.Message, userMessage string) code.Code {
	if session == nil || len(history) > 0 || session.Title != model.DefaultSessionTitle {
		return code.CodeSuccess
	}

	title := strings.TrimSpace(userMessage)
	if title == "" {
		return code.CodeSuccess
	}

	updated, err := sessionDAO.UpdateSessionTitleBySessionIDAndUserRefID(session.SessionID, userRefID, title)
	if err != nil {
		log.Println("syncSessionTitleWithFirstMessage error:", err)
		return code.CodeServerBusy
	}
	if updated {
		session.Title = title
	}

	return code.CodeSuccess
}

func createSession(userRefID uint, title string) (*model.Session, code.Code) {
	newSession := &model.Session{
		SessionID: uuid.New().String(),
		UserRefID: userRefID,
		Title:     title,
	}
	createdSession, err := sessionDAO.CreateSession(newSession)
	if err != nil {
		log.Println("createSession error:", err)
		return nil, code.CodeServerBusy
	}
	return createdSession, code.CodeSuccess
}

func resolveExecutionSession(userRefID uint, sessionID, userMessage string, allowCreateSession bool) (*model.Session, code.Code) {
	if sessionID == "" {
		if !allowCreateSession {
			return nil, code.CodeInvalidParams
		}
		return createSession(userRefID, userMessage)
	}

	return checkOwnedSession(sessionID, userRefID)
}

func prepareChatExecution(
	ctx context.Context,
	userRefID uint,
	sessionID, userMessage string,
	enabledToolNames []string,
	thinkingMode bool,
	allowCreateSession bool,
) (*preparedChatExecution, code.Code) {
	normalizedToolNames := agenttools.NormalizeToolNames(enabledToolNames)
	hasEnabledTools := len(normalizedToolNames) > 0
	agent, err := agentcommon.GetAgentManager().CreateAgentForChat(
		ctx,
		userRefID,
		normalizedToolNames,
		thinkingMode,
		buildAgentInstruction(hasEnabledTools),
	)
	if err != nil {
		if agenttools.IsUnknownToolError(err) {
			log.Println("prepareChatExecution CreateAgentForChat invalid tools:", err)
			return nil, code.CodeInvalidParams
		}
		log.Println("prepareChatExecution CreateAgentForChat error:", err)
		return nil, code.AIModelFail
	}

	session, code_ := resolveExecutionSession(userRefID, sessionID, userMessage, allowCreateSession)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	history, err := loadSessionHistory(session, userRefID)
	if err != nil {
		log.Println("prepareChatExecution loadSessionHistory error:", err)
		return nil, code.CodeServerBusy
	}
	if code_ := syncSessionTitleWithFirstMessage(session, userRefID, history, userMessage); code_ != code.CodeSuccess {
		return nil, code_
	}

	userMessageIndex := len(history)
	userMessageSchema := schema.UserMessage(userMessage)
	persistMessage(session.SessionID, buildStoredMessage(session, userRefID, userMessageIndex, userMessageSchema))

	return &preparedChatExecution{
		session:                    session,
		userRefID:                  userRefID,
		conversation:               buildConversationMessages(history, userMessageSchema),
		firstAssistantMessageIndex: userMessageIndex + 1,
		agent:                      agent,
	}, code.CodeSuccess
}

// Generate 同步生成响应。
func Generate(ctx context.Context, userRefID uint, sessionID, userMessage string, enabledToolNames []string, thinkingMode bool) (*GenerateResult, code.Code) {
	run, code_ := prepareChatExecution(ctx, userRefID, sessionID, userMessage, enabledToolNames, thinkingMode, true)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	produced, finalMessage, err := agentcommon.CollectAgentMessages(ctx, run.agent, run.conversation)
	if err != nil {
		if isRequestCanceled(ctx, err) {
			log.Printf("Generate canceled: %v", err)
			return nil, code.CodeServerBusy
		}
		log.Println("Generate error:", err)
		return nil, code.AIModelFail
	}
	if len(produced) == 0 && finalMessage != nil {
		produced = append(produced, finalMessage)
	}

	lastIndex := persistProducedMessages(run.session, run.userRefID, run.firstAssistantMessageIndex, produced)
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
		SessionID:    run.session.SessionID,
		MessageIndex: lastIndex,
		Message:      finalMessage,
	}, code.CodeSuccess
}

// Stream 打开一个可供 controller 消费的 SSE 事件流。
func Stream(ctx context.Context, userRefID uint, sessionID, userMessage string, enabledToolNames []string, thinkingMode bool) (*StreamHandle, code.Code) {
	run, code_ := prepareChatExecution(ctx, userRefID, sessionID, userMessage, enabledToolNames, thinkingMode, false)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	events := make(chan StreamEnvelope)
	go func() {
		defer close(events)

		if err := emitEvent(ctx, events, NewSuccessEvent(StreamEventTypeResponseCreated, nil)); err != nil {
			return
		}

		produced, err := agentcommon.StreamAgentMessages(ctx, run.agent, run.conversation, &agentcommon.StreamMessageSink{
			OnChunk: func(msg *schema.Message) error {
				return emitEvent(ctx, events, NewSuccessEvent(StreamEventTypeResponseMessageDelta, &StreamDeltaResponse{
					Delta: msg,
				}))
			},
			OnComplete: func(msg *schema.Message) error {
				return emitEvent(ctx, events, NewSuccessEvent(StreamEventTypeResponseMessageCompleted, &StreamCompletedResponse{
					Message: msg,
				}))
			},
		})
		if err != nil {
			if isRequestCanceled(ctx, err) {
				log.Printf("Stream canceled: %v", err)
				return
			}
			log.Println("Stream StreamAgentMessages error:", err)
			_ = emitEvent(ctx, events, NewErrorEvent(code.AIModelFail, "agent stream failed"))
			return
		}

		persistProducedMessages(run.session, run.userRefID, run.firstAssistantMessageIndex, produced)
		_ = emitEvent(ctx, events, NewSuccessEvent(StreamEventTypeResponseDone, nil))
	}()

	return &StreamHandle{
		SessionID: run.session.SessionID,
		Events:    events,
	}, code.CodeSuccess
}

// ListHistoryMessages 获取当前用户的会话消息列表。
func ListHistoryMessages(sessionID string, userRefID uint) ([]HistoryMessageItem, code.Code) {
	session, code_ := checkOwnedSession(sessionID, userRefID)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	msgs, err := loadSessionHistory(session, userRefID)
	if err != nil {
		log.Println("ListHistoryMessages error:", err)
		return nil, code.CodeServerBusy
	}

	return buildHistoryMessageItems(msgs), code.CodeSuccess
}
