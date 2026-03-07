package redis

import (
	"GopherAI/config"
	"fmt"
)

// GenerateCaptcha 生成验证码缓存 key
func GenerateCaptcha(email string) string {
	return fmt.Sprintf(config.DefaultRedisKeyConfig.CaptchaPrefix, email)
}