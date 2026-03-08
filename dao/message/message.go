package message

import (
	"GopherAI/common/postgres"
	"GopherAI/model"
)

// GetMessagesBySessionID 根据会话ID获取消息列表
func GetMessagesBySessionID(sessionID string) ([]model.Message, error) {
	var msgs []model.Message
	err := postgres.DB.Where("session_id = ?", sessionID).Order("created_at asc").Find(&msgs).Error
	return msgs, err
}

// GetMessagesBySessionIDs 根据多个会话ID获取消息列表
func GetMessagesBySessionIDs(sessionIDs []string) ([]model.Message, error) {
	var msgs []model.Message
	if len(sessionIDs) == 0 {
		return msgs, nil
	}
	err := postgres.DB.Where("session_id IN ?", sessionIDs).Order("created_at asc").Find(&msgs).Error
	return msgs, err
}

// CreateMessage 创建消息
func CreateMessage(message *model.Message) (*model.Message, error) {
	err := postgres.DB.Create(message).Error
	return message, err
}

// GetAllMessages 获取所有消息
func GetAllMessages() ([]model.Message, error) {
	var msgs []model.Message
	err := postgres.DB.Order("created_at asc").Find(&msgs).Error
	return msgs, err
}

// GetNextMessageIndex 获取下一条消息的索引
func GetNextMessageIndex(sessionID string) (int, error) {
	var maxIndex int
	err := postgres.DB.Model(&model.Message{}).
		Where("session_id = ?", sessionID).
		Select("COALESCE(MAX(\"index\"), -1)").
		Scan(&maxIndex).Error
	return maxIndex + 1, err
}

// GetMessagesBySessionIDOrdered 按索引获取消息列表
func GetMessagesBySessionIDOrdered(sessionID string) ([]model.Message, error) {
	var msgs []model.Message
	err := postgres.DB.
		Where("session_id = ?", sessionID).
		Order("\"index\" asc").
		Find(&msgs).Error
	return msgs, err
}

// CreateMessageWithIndex 创建带索引的消息
func CreateMessageWithIndex(msg *model.Message) error {
	// 如果没有指定索引，自动获取下一个索引
	if msg.Index == 0 {
		nextIndex, err := GetNextMessageIndex(msg.SessionID)
		if err != nil {
			return err
		}
		msg.Index = nextIndex
	}
	return postgres.DB.Create(msg).Error
}

// GetMessageByIndex 根据索引获取消息
func GetMessageByIndex(sessionID string, index int) (*model.Message, error) {
	var msg model.Message
	err := postgres.DB.
		Where("session_id = ? AND \"index\" = ?", sessionID, index).
		First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}
