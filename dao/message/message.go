package message

import (
	"GopherAI/common/postgres"
	"GopherAI/common/redis"
	"GopherAI/config"
	"GopherAI/model"
	"context"
	"encoding/json"
	"time"

	redisCli "github.com/redis/go-redis/v9"
)

var ctx = context.Background()

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

// ListMessagesBySessionAndUserOrdered 按索引获取指定用户的会话消息列表
func ListMessagesBySessionAndUserOrdered(sessionID string, userName string) ([]model.Message, error) {
	var msgs []model.Message
	err := postgres.DB.
		Where("session_id = ? AND user_name = ?", sessionID, userName).
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

// GetCachedMessages 获取会话消息缓存。
func GetCachedMessages(sessionID string) ([]*model.Message, error) {
	key := redis.GenerateMessageKey(sessionID)

	data, err := redis.Rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redisCli.Nil {
			return []*model.Message{}, nil
		}
		return nil, err
	}

	var messages []*model.Message
	if err := json.Unmarshal([]byte(data), &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// StoreCachedMessages 覆盖写入会话消息缓存。
func StoreCachedMessages(sessionID string, messages []*model.Message) error {
	if len(messages) == 0 {
		return nil
	}

	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	key := redis.GenerateMessageKey(sessionID)
	ttl := time.Duration(config.DefaultRedisKeyConfig.MessageTTL) * time.Hour
	return redis.Rdb.Set(ctx, key, data, ttl).Err()
}

// AppendCachedMessage 追加一条消息到会话缓存。
func AppendCachedMessage(sessionID string, msg *model.Message) error {
	messages, err := GetCachedMessages(sessionID)
	if err != nil {
		return err
	}

	messages = append(messages, msg)
	return StoreCachedMessages(sessionID, messages)
}

// DeleteCachedMessages 删除会话消息缓存。
func DeleteCachedMessages(sessionID string) error {
	key := redis.GenerateMessageKey(sessionID)
	return redis.Rdb.Del(ctx, key).Err()
}

// RefreshCachedMessagesTTL 刷新会话消息缓存的过期时间。
func RefreshCachedMessagesTTL(sessionID string) error {
	key := redis.GenerateMessageKey(sessionID)
	ttl := time.Duration(config.DefaultRedisKeyConfig.MessageTTL) * time.Hour
	return redis.Rdb.Expire(ctx, key, ttl).Err()
}
