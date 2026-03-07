package session

import (
	"GopherAI/common/code"
	"GopherAI/common/llm"
	"GopherAI/common/rabbitmq"
	redis_cache "GopherAI/common/redis"
	"GopherAI/dao/message"
	"GopherAI/dao/session"
	"GopherAI/model"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

var ctx = context.Background()

// SystemPrompt 系统提示词
const SystemPrompt = `你是一个智能助手，可以帮助用户解答各种问题。请用中文回答，保持回答简洁、准确、友好。`

// saveMessageToDB 异步保存消息到数据库（通过 RabbitMQ）
func saveMessageToDB(sessionID string, content string, userName string, isUser bool) {
	data := rabbitmq.GenerateMessageMQParam(sessionID, content, userName, isUser)
	if err := rabbitmq.RMQMessage.Publish(data); err != nil {
		log.Printf("saveMessageToDB error: %v", err)
	}
}

// appendMessageToRedis 追加消息到 Redis 缓存
func appendMessageToRedis(sessionID string, content string, isUser bool) error {
	msg := &model.Message{
		SessionID: sessionID,
		Content:   content,
		IsUser:    isUser,
		CreatedAt: time.Now(),
	}
	return redis_cache.AppendMessage(sessionID, msg)
}

// getMessagesFromRedis 从 Redis 获取消息历史（Cache-Aside 模式）
// 先查 Redis，如果 miss 则从 PostgreSQL 加载并写入 Redis
func getMessagesFromRedis(sessionID string) ([]*model.Message, error) {
	// 1. 先从 Redis 获取
	messages, err := redis_cache.GetMessages(sessionID)
	if err != nil {
		log.Printf("getMessagesFromRedis redis error: %v", err)
		// Redis 出错，降级到数据库
	} else if len(messages) > 0 {
		// Redis 命中，直接返回
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

	// 4. 写入 Redis 缓存（异步，不阻塞）
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
func buildMessages(history []*model.Message, userQuestion string) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history)+2)

	// 添加系统提示词
	messages = append(messages, &schema.Message{
		Role:    schema.System,
		Content: SystemPrompt,
	})

	// 添加历史消息
	for _, m := range history {
		role := schema.Assistant
		if m.IsUser {
			role = schema.User
		}
		messages = append(messages, &schema.Message{
			Role:    role,
			Content: m.Content,
		})
	}

	// 添加当前用户问题
	messages = append(messages, &schema.Message{
		Role:    schema.User,
		Content: userQuestion,
	})

	return messages
}

func GetUserSessionsByUserName(userName string) ([]model.SessionInfo, error) {
	sessions, err := session.GetSessionsByUserName(userName)
	if err != nil {
		log.Println("GetUserSessionsByUserName error:", err)
		return nil, err
	}

	log.Printf("GetUserSessionsByUserName: found %d sessions for user %s\n", len(sessions), userName)

	var SessionInfos []model.SessionInfo
	for _, s := range sessions {
		log.Printf("Session: ID=%s, Title=%s\n", s.ID, s.Title)
		SessionInfos = append(SessionInfos, model.SessionInfo{
			SessionID: s.ID,
			Title:     s.Title,
		})
	}

	return SessionInfos, nil
}

func CreateSessionAndSendMessage(userName string, userQuestion string, modelType string) (string, string, code.Code) {
	// 1：创建一个新的会话
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    userQuestion,
	}
	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateSessionAndSendMessage CreateSession error:", err)
		return "", "", code.CodeServerBusy
	}

	// 2：获取 LLM 客户端
	client := llm.GetLLMClient()

	// 3：构建消息（新会话只有系统提示词和用户问题）
	messages := buildMessages([]*model.Message{}, userQuestion)

	// 4：调用 LLM 生成响应
	aiResponse, err := client.Generate(ctx, messages)
	if err != nil {
		log.Println("CreateSessionAndSendMessage Generate error:", err)
		return "", "", code.AIModelFail
	}

	// 5：保存消息到 Redis 和数据库
	_ = appendMessageToRedis(createdSession.ID, userQuestion, true)
	saveMessageToDB(createdSession.ID, userQuestion, userName, true)

	_ = appendMessageToRedis(createdSession.ID, aiResponse.Content, false)
	saveMessageToDB(createdSession.ID, aiResponse.Content, userName, false)

	return createdSession.ID, aiResponse.Content, code.CodeSuccess
}

func CreateStreamSessionOnly(userName string, userQuestion string) (string, code.Code) {
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    userQuestion,
	}
	createdSession, err := session.CreateSession(newSession)
	if err != nil {
		log.Println("CreateStreamSessionOnly CreateSession error:", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

func StreamMessageToExistingSession(userName string, sessionID string, userQuestion string, modelType string, writer http.ResponseWriter) code.Code {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		log.Println("StreamMessageToExistingSession: streaming unsupported")
		return code.CodeServerBusy
	}

	// 1：获取 LLM 客户端
	client := llm.GetLLMClient()

	// 2：从 Redis 获取消息历史
	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("StreamMessageToExistingSession getMessagesFromRedis error:", err)
		return code.CodeServerBusy
	}

	// 3：构建消息
	messages := buildMessages(history, userQuestion)

	// 4：先保存用户消息到 Redis 和数据库
	_ = appendMessageToRedis(sessionID, userQuestion, true)
	saveMessageToDB(sessionID, userQuestion, userName, true)

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

	// 7：保存 AI 响应到 Redis 和数据库
	_ = appendMessageToRedis(sessionID, fullResponse, false)
	saveMessageToDB(sessionID, fullResponse, userName, false)

	return code.CodeSuccess
}

func CreateStreamSessionAndSendMessage(userName string, userQuestion string, modelType string, writer http.ResponseWriter) (string, code.Code) {
	sessionID, code_ := CreateStreamSessionOnly(userName, userQuestion)
	if code_ != code.CodeSuccess {
		return "", code_
	}

	code_ = StreamMessageToExistingSession(userName, sessionID, userQuestion, modelType, writer)
	if code_ != code.CodeSuccess {
		return sessionID, code_
	}

	return sessionID, code.CodeSuccess
}

func ChatSend(userName string, sessionID string, userQuestion string, modelType string) (string, code.Code) {
	// 1：获取 LLM 客户端
	client := llm.GetLLMClient()

	// 2：从 Redis 获取消息历史
	history, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("ChatSend getMessagesFromRedis error:", err)
		return "", code.CodeServerBusy
	}

	// 3：构建消息
	messages := buildMessages(history, userQuestion)

	// 4：先保存用户消息到 Redis 和数据库
	_ = appendMessageToRedis(sessionID, userQuestion, true)
	saveMessageToDB(sessionID, userQuestion, userName, true)

	// 5：调用 LLM 生成响应
	aiResponse, err := client.Generate(ctx, messages)
	if err != nil {
		log.Println("ChatSend Generate error:", err)
		return "", code.AIModelFail
	}

	// 6：保存 AI 响应到 Redis 和数据库
	_ = appendMessageToRedis(sessionID, aiResponse.Content, false)
	saveMessageToDB(sessionID, aiResponse.Content, userName, false)

	return aiResponse.Content, code.CodeSuccess
}

func GetChatHistory(userName string, sessionID string) ([]model.History, code.Code) {
	messages, err := getMessagesFromRedis(sessionID)
	if err != nil {
		log.Println("GetChatHistory getMessagesFromRedis error:", err)
		return nil, code.CodeServerBusy
	}

	history := make([]model.History, 0, len(messages))
	for _, msg := range messages {
		history = append(history, model.History{
			IsUser:  msg.IsUser,
			Content: msg.Content,
		})
	}

	return history, code.CodeSuccess
}

func ChatStreamSend(userName string, sessionID string, userQuestion string, modelType string, writer http.ResponseWriter) code.Code {
	return StreamMessageToExistingSession(userName, sessionID, userQuestion, modelType, writer)
}

func DeleteSession(userName string, sessionID string) code.Code {
	// 1: 从 Redis 中删除消息缓存
	_ = redis_cache.DeleteMessages(sessionID)

	// 2: 从数据库中删除会话
	err := session.DeleteSession(sessionID, userName)
	if err != nil {
		log.Println("DeleteSession error:", err)
		return code.CodeServerBusy
	}

	return code.CodeSuccess
}

func UpdateSessionTitle(userName string, sessionID string, title string) code.Code {
	log.Printf("UpdateSessionTitle: userName=%s, sessionID=%s, title=%s\n", userName, sessionID, title)
	err := session.UpdateSessionTitle(sessionID, userName, title)
	if err != nil {
		log.Println("UpdateSessionTitle error:", err)
		return code.CodeServerBusy
	}
	return code.CodeSuccess
}