package agent

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"GopherAI/common/agent"
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
	SSEEventTypeContentDelta = "content_delta"
	SSEEventTypeMessageEnd   = "message_end"
)

// SSEEvent SSE 事件结构
type SSEEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// MetaEvent 元信息事件
type MetaEvent struct {
	SessionID    string `json:"session_id"`
	MessageIndex int    `json:"message_index"`
}

// ToolCallEvent 工具调用事件
type ToolCallEvent struct {
	ToolID    string          `json:"tool_id"`
	Function  string          `json:"function"`
	Arguments json.RawMessage `json:"arguments"`
}

// ContentDeltaEvent 内容增量事件
type ContentDeltaEvent struct {
	Content string `json:"content"`
}

// MessageEndEvent 消息结束事件
type MessageEndEvent struct {
	Status string `json:"status"` // "completed" | "error"
}

// AgentResult Agent 执行结果
type AgentResult struct {
	SessionID    string           `json:"session_id"`
	MessageIndex int              `json:"message_index"`
	Role         string           `json:"role"`
	Content      string           `json:"content"`
	ToolCalls    []model.ToolCall `json:"tool_calls,omitempty"`
}

// saveMessageToDB 异步保存消息到数据库（通过 RabbitMQ）
func saveMessageToDB(sessionID string, content string, userName string, role string, toolCalls []model.ToolCall) {
	data := rabbitmq.GenerateMessageMQParam(sessionID, content, userName, role)
	if err := rabbitmq.RMQMessage.Publish(data); err != nil {
		log.Printf("saveMessageToDB error: %v", err)
	}
}

// saveMessageToDBNew 异步保存消息到数据库（新格式）
func saveMessageToDBNew(msg *model.Message) {
	data := rabbitmq.GenerateMessageMQParam(msg.SessionID, msg.Content, msg.UserName, msg.Role)
	if err := rabbitmq.RMQMessage.Publish(data); err != nil {
		log.Printf("saveMessageToDBNew error: %v", err)
	}
}

// appendMessageToRedis 追加消息到 Redis 缓存
func appendMessageToRedis(sessionID string, content string, role string, index int) error {
	msg := &model.Message{
		SessionID: sessionID,
		Content:   content,
		Role:      role,
		Index:     index,
		CreatedAt: time.Now(),
	}
	return redis_cache.AppendMessage(sessionID, msg)
}

// getMessagesFromRedis 从 Redis 获取消息历史（Cache-Aside 模式）
func getMessagesFromRedis(sessionID string) ([]*model.Message, error) {
	// 1. 先从 Redis 获取
	messages, err := redis_cache.GetMessages(sessionID)
	if err != nil {
		log.Printf("getMessagesFromRedis redis error: %v", err)
	} else if len(messages) > 0 {
		return messages, nil
	}

	// 2. Redis miss 或出错，从 PostgreSQL 加载
	dbMessages, err := message.GetMessagesBySessionIDOrdered(sessionID)
	if err != nil {
		log.Printf("getMessagesFromRedis db error: %v", err)
		return nil, err
	}

	// 3. 转换为指针数组
	result := make([]*model.Message, 0, len(dbMessages))
	for i := range dbMessages {
		result = append(result, &dbMessages[i])
	}

	// 4. 写入 Redis 缓存（异步）
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

	// 添加系统提示词
	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: SystemPrompt,
	})

	// 添加历史消息
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

	// 添加当前用户内容
	if userContent != "" {
		messages = append(messages, &schema.Message{
			Role:    schema.User,
			Content: userContent,
		})
	}

	return messages
}

// sendSSEEvent 发送 SSE 事件
func sendSSEEvent(writer http.ResponseWriter, eventType string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	event := SSEEvent{
		Type:    eventType,
		Payload: data,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte("data: " + string(eventData) + "\n\n"))
	if err != nil {
		return err
	}

	if flusher, ok := writer.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// sendSSEError 发送 SSE 错误事件
func sendSSEError(writer http.ResponseWriter, code_ code.Code) {
	sendSSEEvent(writer, SSEEventTypeMessageEnd, MessageEndEvent{
		Status: "error",
	})
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
func Generate(userName string, sessionID string, userContent string, toolNames []string) (*AgentResult, code.Code) {
	return GenerateWithContext(ctx, userName, sessionID, userContent, toolNames)
}

// GenerateWithContext 带上下文的同步生成响应
func GenerateWithContext(ctx context.Context, userName string, sessionID string, userContent string, toolNames []string) (*AgentResult, code.Code) {
	// 1：获取 Agent 管理器
	agentMgr := agent.GetAgentManager()

	// 2：从 Redis 获取消息历史
	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("Generate getMessagesFromRedis error:", err)
		return nil, code.CodeServerBusy
	}

	// 3：计算下一个索引
	nextIndex := len(history)

	// 4：保存用户消息
	userMsg := &model.Message{
		SessionID: sessionID,
		UserName:  userName,
		Index:     nextIndex,
		Role:      "user",
		Content:   userContent,
		CreatedAt: time.Now(),
	}
	_ = appendMessageToRedis(sessionID, userContent, "user", nextIndex)
	saveMessageToDBNew(userMsg)

	// 5：构建消息
	messages := buildMessages(history, userContent)

	// 6：调用 Agent 生成响应
	aiResponse, err := agentMgr.GenerateWithTools(ctx, sessionID, userName, messages, toolNames)
	if err != nil {
		log.Println("Generate error:", err)
		return nil, code.AIModelFail
	}

	// 7：保存 AI 响应
	aiIndex := nextIndex + 1
	_ = appendMessageToRedis(sessionID, aiResponse.Content, "assistant", aiIndex)
	saveMessageToDB(sessionID, aiResponse.Content, userName, "assistant", nil)

	return &AgentResult{
		SessionID:    sessionID,
		MessageIndex: aiIndex,
		Role:         "assistant",
		Content:      aiResponse.Content,
	}, code.CodeSuccess
}

// Stream 流式生成响应
func Stream(userName string, sessionID string, userContent string, toolNames []string, writer http.ResponseWriter) code.Code {
	return StreamWithContext(ctx, userName, sessionID, userContent, toolNames, writer)
}

// StreamWithContext 带上下文的流式生成响应
func StreamWithContext(ctx context.Context, userName string, sessionID string, userContent string, toolNames []string, writer http.ResponseWriter) code.Code {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		log.Println("Stream: streaming unsupported")
		return code.CodeServerBusy
	}
	_ = flusher // 避免未使用警告

	// 1：获取 Agent 管理器
	agentMgr := agent.GetAgentManager()

	// 2：从 Redis 获取消息历史
	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("Stream getMessagesFromRedis error:", err)
		return code.CodeServerBusy
	}

	// 3：计算索引
	nextIndex := len(history)
	aiIndex := nextIndex + 1

	// 4：发送 meta 事件
	sendSSEEvent(writer, SSEEventTypeMeta, MetaEvent{
		SessionID:    sessionID,
		MessageIndex: aiIndex,
	})

	// 5：保存用户消息
	userMsg := &model.Message{
		SessionID: sessionID,
		UserName:  userName,
		Index:     nextIndex,
		Role:      "user",
		Content:   userContent,
		CreatedAt: time.Now(),
	}
	_ = appendMessageToRedis(sessionID, userContent, "user", nextIndex)
	saveMessageToDBNew(userMsg)

	// 6：构建消息
	messages := buildMessages(history, userContent)

	// 7：流式调用 Agent
	var fullResponse string
	var toolCalls []model.ToolCall

	contentCb := func(content string) {
		fullResponse += content
		sendSSEEvent(writer, SSEEventTypeContentDelta, ContentDeltaEvent{
			Content: content,
		})
	}

	toolCallCb := func(tc model.ToolCall) {
		toolCalls = append(toolCalls, tc)
		sendSSEEvent(writer, SSEEventTypeToolCall, ToolCallEvent{
			ToolID:    tc.ToolID,
			Function:  tc.Function,
			Arguments: tc.Arguments,
		})
	}

	fullResponse, toolCalls, err = agentMgr.StreamWithCallbacks(ctx, sessionID, userName, messages, toolNames, contentCb, toolCallCb)
	if err != nil {
		log.Println("Stream error:", err)
		sendSSEError(writer, code.AIModelFail)
		return code.AIModelFail
	}

	// 8：发送结束事件
	sendSSEEvent(writer, SSEEventTypeMessageEnd, MessageEndEvent{
		Status: "completed",
	})

	// 9：保存 AI 响应
	_ = appendMessageToRedis(sessionID, fullResponse, "assistant", aiIndex)
	saveMessageToDB(sessionID, fullResponse, userName, "assistant", toolCalls)

	return code.CodeSuccess
}

// StreamWithMeta 流式生成响应（带会话创建）
func StreamWithMeta(userName string, sessionID string, userContent string, toolNames []string, writer http.ResponseWriter) (*AgentResult, code.Code) {
	// 如果没有 session_id，先创建
	if sessionID == "" {
		var code_ code.Code
		sessionID, code_ = CreateSessionOnly(userName, userContent)
		if code_ != code.CodeSuccess {
			return nil, code_
		}
	}

	code_ := StreamWithContext(ctx, userName, sessionID, userContent, toolNames, writer)
	if code_ != code.CodeSuccess {
		return nil, code_
	}

	return &AgentResult{
		SessionID: sessionID,
	}, code.CodeSuccess
}

// CreateSessionAndStream 创建新会话并流式响应
func CreateSessionAndStream(userName string, userContent string, toolNames []string, writer http.ResponseWriter) (string, code.Code) {
	sessionID, code_ := CreateSessionOnly(userName, userContent)
	if code_ != code.CodeSuccess {
		return "", code_
	}

	code_ = Stream(userName, sessionID, userContent, toolNames, writer)
	if code_ != code.CodeSuccess {
		return sessionID, code_
	}

	return sessionID, code.CodeSuccess
}

// Regenerate 重新生成响应
func Regenerate(ctx context.Context, userName string, sessionID string, fromIndex int, toolNames []string, writer http.ResponseWriter, stream bool) (*AgentResult, code.Code) {
	// 1. 验证会话存在
	sess, err := session.GetSessionByID(sessionID)
	if err != nil || sess == nil {
		return nil, code.CodeSessionNotFound
	}

	// 2. 验证索引有效性
	count, err := message.GetMessageCount(sessionID)
	if err != nil {
		return nil, code.CodeServerBusy
	}
	if fromIndex < 0 || fromIndex >= count {
		return nil, code.CodeInvalidParams
	}

	// 3. 截断消息
	err = message.TruncateMessages(sessionID, fromIndex)
	if err != nil {
		return nil, code.CodeServerBusy
	}

	// 4. 清除 Redis 缓存
	redis_cache.DeleteMessages(sessionID)

	// 5. 清除 Agent 缓存
	agent.GetAgentManager().ClearAgent(sessionID)

	// 6. 获取截断后的历史
	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		return nil, code.CodeServerBusy
	}

	// 7. 获取最后一条用户消息
	var lastUserContent string
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role == "user" {
			lastUserContent = history[i].Content
			break
		}
	}

	if lastUserContent == "" {
		return nil, code.CodeInvalidParams
	}

	// 8. 重新生成响应
	if stream {
		return StreamWithMeta(userName, sessionID, lastUserContent, toolNames, writer)
	}
	return GenerateWithContext(ctx, userName, sessionID, lastUserContent, toolNames)
}

// GetMessages 获取会话消息列表
func GetMessages(sessionID string, userName string) ([]model.Message, error) {
	return message.GetMessagesBySessionIDOrdered(sessionID)
}