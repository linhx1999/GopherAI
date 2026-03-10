package redis

import (
	"GopherAI/config"
	"strconv"

	redisCli "github.com/redis/go-redis/v9"
)

var Rdb *redisCli.Client

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
