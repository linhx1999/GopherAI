package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type MainConfig struct {
	Port    int
	AppName string
	Host    string
}

type EmailConfig struct {
	Authcode string
	Email    string
}

type RedisConfig struct {
	RedisPort     int
	RedisDb       int
	RedisHost     string
	RedisPassword string
}

type PostgresConfig struct {
	PostgresPort     int
	PostgresHost     string
	PostgresUser     string
	PostgresPassword string
	PostgresDatabase string
	PostgresSSLMode  string
}

type JwtConfig struct {
	ExpireDuration int
	Issuer         string
	Subject        string
	Key            string
}

type Rabbitmq struct {
	RabbitmqPort     int
	RabbitmqHost     string
	RabbitmqUsername string
	RabbitmqPassword string
	RabbitmqVhost    string
}

type RagModelConfig struct {
	RagEmbeddingModel string
	RagChatModelName  string
	RagDocDir         string
	RagBaseUrl        string
	RagDimension      int
}

type VoiceServiceConfig struct {
	VoiceServiceApiKey    string
	VoiceServiceSecretKey string
}

type OpenAIConfig struct {
	ApiKey             string
	ModelName          string
	ReasoningModelName string
	BaseUrl            string
}

type MinioConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type Config struct {
	EmailConfig
	RedisConfig
	PostgresConfig
	JwtConfig
	MainConfig
	Rabbitmq
	RagModelConfig
	VoiceServiceConfig
	OpenAIConfig
	MinioConfig
}

type RedisKeyConfig struct {
	CaptchaPrefix string
	MessagePrefix string // 消息历史缓存 key 前缀
	MessageTTL    int    // 消息缓存过期时间（小时）
}

var DefaultRedisKeyConfig = RedisKeyConfig{
	CaptchaPrefix: "captcha:%s",
	MessagePrefix: "chat:messages:%s", // chat:messages:{sessionID}
	MessageTTL:    2,                  // 2小时过期
}

var config *Config

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// getEnvInt 获取整数类型环境变量
func getEnvInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

// InitConfig 初始化项目配置
func InitConfig() error {
	// 加载 .env 文件（如果存在）
	_ = godotenv.Load(".env")

	// 从环境变量读取所有配置
	config = &Config{
		MainConfig: MainConfig{
			Port:    getEnvInt("APP_PORT", 9090),
			AppName: getEnv("APP_NAME", "GopherAI"),
			Host:    getEnv("APP_HOST", "0.0.0.0"),
		},
		EmailConfig: EmailConfig{
			Authcode: getEnv("EMAIL_AUTHCODE", ""),
			Email:    getEnv("EMAIL_FROM", ""),
		},
		RedisConfig: RedisConfig{
			RedisHost:     getEnv("REDIS_HOST", "127.0.0.1"),
			RedisPort:     getEnvInt("REDIS_PORT", 6379),
			RedisPassword: getEnv("REDIS_PASSWORD", ""),
			RedisDb:       getEnvInt("REDIS_DB", 0),
		},
		PostgresConfig: PostgresConfig{
			PostgresHost:     getEnv("POSTGRES_HOST", "127.0.0.1"),
			PostgresPort:     getEnvInt("POSTGRES_PORT", 5432),
			PostgresUser:     getEnv("POSTGRES_USER", "postgres"),
			PostgresPassword: getEnv("POSTGRES_PASSWORD", ""),
			PostgresDatabase: getEnv("POSTGRES_DB", "gopherai"),
			PostgresSSLMode:  getEnv("POSTGRES_SSL_MODE", "disable"),
		},
		JwtConfig: JwtConfig{
			ExpireDuration: getEnvInt("JWT_EXPIRE_DURATION", 8760),
			Issuer:         getEnv("JWT_ISSUER", "huanheart"),
			Subject:        getEnv("JWT_SUBJECT", "GopherAI"),
			Key:            getEnv("JWT_SECRET_KEY", "GopherAI-v1"),
		},
		Rabbitmq: Rabbitmq{
			RabbitmqHost:     getEnv("RABBITMQ_HOST", "localhost"),
			RabbitmqPort:     getEnvInt("RABBITMQ_PORT", 5672),
			RabbitmqUsername: getEnv("RABBITMQ_DEFAULT_USER", ""),
			RabbitmqPassword: getEnv("RABBITMQ_DEFAULT_PASS", ""),
			RabbitmqVhost:    getEnv("RABBITMQ_VHOST", "/"),
		},
		RagModelConfig: RagModelConfig{
			RagEmbeddingModel: getEnv("RAG_EMBEDDING_MODEL", "text-embedding-v4"),
			RagChatModelName:  getEnv("RAG_CHAT_MODEL", "qwen-turbo"),
			RagDocDir:         getEnv("RAG_DOC_DIR", "./docs"),
			RagBaseUrl:        getEnv("RAG_BASE_URL", "https://dashscope.aliyuncs.com/compatible-mode/v1"),
			RagDimension:      getEnvInt("RAG_DIMENSION", 1024),
		},
		VoiceServiceConfig: VoiceServiceConfig{
			VoiceServiceApiKey:    getEnv("VOICE_API_KEY", ""),
			VoiceServiceSecretKey: getEnv("VOICE_SECRET_KEY", ""),
		},
		OpenAIConfig: OpenAIConfig{
			ApiKey:             getEnv("OPENAI_API_KEY", ""),
			ModelName:          getEnv("OPENAI_MODEL_NAME", ""),
			ReasoningModelName: getEnv("OPENAI_REASONING_MODEL_NAME", ""),
			BaseUrl:            getEnv("OPENAI_BASE_URL", ""),
		},
		MinioConfig: MinioConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", ""),
			SecretKey: getEnv("MINIO_SECRET_KEY", ""),
			Bucket:    getEnv("MINIO_BUCKET", "gopherai"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		},
	}

	return nil
}

func GetConfig() *Config {
	if config == nil {
		config = new(Config)
		_ = InitConfig()
	}
	return config
}
