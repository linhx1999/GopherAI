package main

import (
	commondeep "GopherAI/common/deepagent"
	"GopherAI/common/llm"
	"GopherAI/common/minio"
	"GopherAI/common/postgres"
	"GopherAI/common/rabbitmq"
	"GopherAI/common/redis"
	"GopherAI/config"
	"GopherAI/router"
	"context"
	"fmt"
	"log"
)

func StartServer(addr string, port int) error {
	r := router.InitRouter()
	return r.Run(fmt.Sprintf("%s:%d", addr, port))
}

func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port

	// 初始化 PostgreSQL
	if err := postgres.InitPostgres(); err != nil {
		log.Println("InitPostgres error , " + err.Error())
		return
	}
	log.Println("postgres init success")

	// 初始化 Redis（用于验证码缓存和消息历史缓存）
	redis.Init()
	log.Println("redis init success")

	// 初始化 RabbitMQ
	rabbitmq.InitRabbitMQ()
	log.Println("rabbitmq init success")

	// 初始化 MinIO
	if err := minio.InitMinIO(); err != nil {
		log.Println("InitMinIO error, " + err.Error())
		return
	}
	log.Println("minio init success")

	// 初始化 LLM 客户端
	client := llm.GetLLMClient()
	if err := client.Init(context.Background()); err != nil {
		log.Println("InitLLM error, " + err.Error())
		// 不阻止服务启动，允许后续重试
	} else {
		log.Println("llm client init success")
	}

	if commondeep.FeatureEnabled() {
		if err := commondeep.Init(context.Background()); err != nil {
			log.Println("InitDeepAgent error, " + err.Error())
		} else {
			commondeep.StartReaper(context.Background())
			log.Println("deep agent runtime init success")
		}
	}

	err := StartServer(host, port)
	if err != nil {
		panic(err)
	}
}
