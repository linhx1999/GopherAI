package redis

import (
	"GopherAI/config"
	"GopherAI/model"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	redisCli "github.com/redis/go-redis/v9"
)

var Rdb *redisCli.Client

var ctx = context.Background()

// Init 初始化 Redis 客户端（用于验证码缓存和消息历史缓存）
func Init() {
	conf := config.GetConfig()
	host := conf.RedisConfig.RedisHost
	port := conf.RedisConfig.RedisPort
	password := conf.RedisConfig.RedisPassword
	db := conf.RedisDb
	addr := host + ":" + strconv.Itoa(port)

	Rdb = redisCli.NewClient(&redisCli.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
		Protocol: 2, // 使用 Protocol 2 避免 maint_notifications 警告
	})
}

// SetCaptchaForEmail 设置邮箱验证码
func SetCaptchaForEmail(email, captcha string) error {
	key := GenerateCaptcha(email)
	expire := 2 * time.Minute
	return Rdb.Set(ctx, key, captcha, expire).Err()
}

// CheckCaptchaForEmail 验证邮箱验证码
func CheckCaptchaForEmail(email, userInput string) (bool, error) {
	key := GenerateCaptcha(email)

	storedCaptcha, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redisCli.Nil {
			return false, nil
		}
		return false, err
	}

	if strings.EqualFold(storedCaptcha, userInput) {
		// 验证成功后删除 key
		_ = Rdb.Del(ctx, key).Err()
		return true, nil
	}

	return false, nil
}

// ==================== 消息历史缓存 ====================

// CacheMessages 缓存会话的消息历史
func CacheMessages(sessionID string, messages []*model.Message) error {
	key := GenerateMessageKey(sessionID)
	if len(messages) == 0 {
		return nil
	}

	// 序列化消息列表
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	// 设置缓存，带过期时间
	ttl := time.Duration(config.DefaultRedisKeyConfig.MessageTTL) * time.Hour
	return Rdb.Set(ctx, key, data, ttl).Err()
}

// GetMessages 获取会话的消息历史
func GetMessages(sessionID string) ([]*model.Message, error) {
	key := GenerateMessageKey(sessionID)

	data, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redisCli.Nil {
			// 缓存不存在，返回空列表
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

// AppendMessage 追加消息到缓存
func AppendMessage(sessionID string, msg *model.Message) error {
	// 获取现有消息
	messages, err := GetMessages(sessionID)
	if err != nil {
		return err
	}

	// 追加新消息
	messages = append(messages, msg)

	// 重新缓存
	return CacheMessages(sessionID, messages)
}

// DeleteMessages 删除会话的消息缓存
func DeleteMessages(sessionID string) error {
	key := GenerateMessageKey(sessionID)
	return Rdb.Del(ctx, key).Err()
}

// RefreshMessageTTL 刷新消息缓存的过期时间
func RefreshMessageTTL(sessionID string) error {
	key := GenerateMessageKey(sessionID)
	ttl := time.Duration(config.DefaultRedisKeyConfig.MessageTTL) * time.Hour
	return Rdb.Expire(ctx, key, ttl).Err()
}