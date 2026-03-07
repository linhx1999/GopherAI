package session

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"GopherAI/common/code"
	"GopherAI/common/llm"
	"GopherAI/common/rabbitmq"
	redis_cache "GopherAI/common/redis"
	"GopherAI/dao/message"
	"GopherAI/dao/session"
	"GopherAI/model"
)

var ctx = context.Background()

// SystemPrompt 系统提示词
const SystemPrompt = `你是一个智能助手，可以帮助用户解答各种问题。请用中文回答，保持回答简洁、准确、友好。`

// saveMessageToDB 异步保存消息到数据库（通过 RabbitMQ）
func saveMessageToDB(sessionID string, content string, userName string, role string) {
	data := rabbitmq.GenerateMessageMQParam(sessionID, content, userName, role)
	if err := rabbitmq.RMQMessage.Publish(data); err != nil {
		log.Printf("saveMessageToDB error: %v", err)
	}
}

// appendMessageToRedis 追加消息到 Redis 缓存
func appendMessageToRedis(sessionID string, content string, role string) error {
	msg := &model.Message{
		SessionID: sessionID,
		Content:   content,
		Role:      role,
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
	dbMessages, err := message.GetMessagesBySessionID(sessionID)
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

// buildMessages 构建发送给 LLM 的消息列表
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
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: userContent,
	})

	return messages
}

// GetUserSessionsByUserName 获取用户会话列表
func GetUserSessionsByUserName(userName string) ([]model.SessionInfo, error) {
	sessions, err := session.GetSessionsByUserName(userName)
	if err != nil {
		log.Println("GetUserSessionsByUserName error:", err)
		return nil, err
	}

	var SessionInfos []model.SessionInfo
	for _, s := range sessions {
		SessionInfos = append(SessionInfos, model.SessionInfo{
			SessionID: s.ID,
			Title:     s.Title,
		})
	}

	return SessionInfos, nil
}

// CreateSessionAndSendMessage 创建新会话并发送消息（同步）
func CreateSessionAndSendMessage(userName string, userContent string) (string, string, code.Code) {
	// 1：创建新会话
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    userContent,
	}
	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateSessionAndSendMessage CreateSession error:", err)
		return "", "", code.CodeServerBusy
	}

	// 2：获取 LLM 客户端
	client := llm.GetLLMClient()
	if client == nil {
		return "", "", code.AIModelFail
	}

	// 3：构建消息
	messages := buildMessages([]*model.Message{}, userContent)

	// 4：调用 LLM 生成响应
	aiResponse, err := client.Generate(ctx, messages)
	if err != nil {
		log.Println("CreateSessionAndSendMessage Generate error:", err)
		return "", "", code.AIModelFail
	}

	// 5：保存消息
	_ = appendMessageToRedis(createdSession.ID, userContent, "user")
	saveMessageToDB(createdSession.ID, userContent, userName, "user")

	_ = appendMessageToRedis(createdSession.ID, aiResponse.Content, "assistant")
	saveMessageToDB(createdSession.ID, aiResponse.Content, userName, "assistant")

	return createdSession.ID, aiResponse.Content, code.CodeSuccess
}

// CreateStreamSessionOnly 仅创建会话（用于流式响应）
func CreateStreamSessionOnly(userName string, title string) (string, code.Code) {
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    title,
	}
	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateStreamSessionOnly error:", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

// StreamMessageToExistingSession 流式发送消息到已有会话
func StreamMessageToExistingSession(userName string, sessionID string, userContent string, writer http.ResponseWriter) code.Code {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		log.Println("StreamMessageToExistingSession: streaming unsupported")
		return code.CodeServerBusy
	}

	// 1：获取 LLM 客户端
	client := llm.GetLLMClient()
	if client == nil {
		return code.AIModelFail
	}

	// 2：获取消息历史
	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("StreamMessageToExistingSession getMessagesFromRedis error:", err)
		return code.CodeServerBusy
	}

	// 3：构建消息
	messages := buildMessages(history, userContent)

	// 4：保存用户消息
	_ = appendMessageToRedis(sessionID, userContent, "user")
	saveMessageToDB(sessionID, userContent, userName, "user")

	// 5：流式调用 LLM
	var fullResponse string
	cb := func(msg string) {
		log.Printf("[SSE] Sending chunk: %s (len=%d)\n", msg, len(msg))
		_, err := writer.Write([]byte("data: " + msg + "\n\n"))
		if err != nil {
			log.Println("[SSE] Write error:", err)
			return
		}
		flusher.Flush()
	}

	fullResponse, err = client.Stream(ctx, messages, cb)
	if err != nil {
		log.Println("StreamMessageToExistingSession Stream error:", err)
		return code.AIModelFail
	}

	// 6：发送完成信号
	_, err = writer.Write([]byte("data: [DONE]\n\n"))
	if err != nil {
		log.Println("StreamMessageToExistingSession write DONE error:", err)
		return code.CodeServerBusy
	}
	flusher.Flush()

	// 7：保存 AI 响应
	_ = appendMessageToRedis(sessionID, fullResponse, "assistant")
	saveMessageToDB(sessionID, fullResponse, userName, "assistant")

	return code.CodeSuccess
}

// ChatSend 同步发送消息到已有会话
func ChatSend(userName string, sessionID string, userContent string) (string, code.Code) {
	// 1：获取 LLM 客户端
	client := llm.GetLLMClient()
	if client == nil {
		return "", code.AIModelFail
	}

	// 2：获取消息历史
	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("ChatSend getMessagesFromRedis error:", err)
		return "", code.CodeServerBusy
	}

	// 3：构建消息
	messages := buildMessages(history, userContent)

	// 4：保存用户消息
	_ = appendMessageToRedis(sessionID, userContent, "user")
	saveMessageToDB(sessionID, userContent, userName, "user")

	// 5：调用 LLM 生成响应
	aiResponse, err := client.Generate(ctx, messages)
	if err != nil {
		log.Println("ChatSend Generate error:", err)
		return "", code.AIModelFail
	}

	// 6：保存 AI 响应
	_ = appendMessageToRedis(sessionID, aiResponse.Content, "assistant")
	saveMessageToDB(sessionID, aiResponse.Content, userName, "assistant")

	return aiResponse.Content, code.CodeSuccess
}

// GetChatHistory 获取聊天历史
func GetChatHistory(userName string, sessionID string) ([]*model.Message, code.Code) {
	messages, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("GetChatHistory error:", err)
		return nil, code.CodeServerBusy
	}

	return messages, code.CodeSuccess
}

// ChatStreamSend 流式发送消息
func ChatStreamSend(userName string, sessionID string, userContent string, writer http.ResponseWriter) code.Code {
	return StreamMessageToExistingSession(userName, sessionID, userContent, writer)
}

// DeleteSession 删除会话
func DeleteSession(userName string, sessionID string) code.Code {
	_ = redis_cache.DeleteMessages(sessionID)

	err := session.DeleteSession(sessionID, userName)
	if err != nil {
		log.Println("DeleteSession error:", err)
		return code.CodeServerBusy
	}

	return code.CodeSuccess
}

// UpdateSessionTitle 更新会话标题
func UpdateSessionTitle(userName string, sessionID string, title string) code.Code {
	err := session.UpdateSessionTitle(sessionID, userName, title)
	if err != nil {
		log.Println("UpdateSessionTitle error:", err)
		return code.CodeServerBusy
	}
	return code.CodeSuccess
}
