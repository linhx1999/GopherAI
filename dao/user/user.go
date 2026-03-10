package user

import (
	"GopherAI/common/postgres"
	redis_cache "GopherAI/common/redis"
	"GopherAI/model"
	"GopherAI/utils"
	"context"
	"strings"
	"time"

	redisCli "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	CodeMsg     = "GopherAI验证码如下(验证码仅限于2分钟有效): "
	UserNameMsg = "GopherAI的账号如下，请保留好，后续可以用账号进行登录 "
)

var ctx = context.Background()

// 这边只能通过账号进行登录
func IsExistUser(username string) (bool, *model.User) {

	user, err := postgres.GetUserByUsername(username)

	if err == gorm.ErrRecordNotFound || user == nil {
		return false, nil
	}

	return true, user
}

func Register(username, email, password string) (*model.User, bool) {
	if user, err := postgres.InsertUser(&model.User{
		Email:    email,
		Name:     username,
		Username: username,
		Password: utils.MD5(password),
	}); err != nil {
		return nil, false
	} else {
		return user, true
	}
}

// StoreEmailCaptcha 写入邮箱验证码缓存。
func StoreEmailCaptcha(email, captcha string) error {
	key := redis_cache.GenerateCaptcha(email)
	return redis_cache.Rdb.Set(ctx, key, captcha, 2*time.Minute).Err()
}

// VerifyEmailCaptcha 验证邮箱验证码，成功后消费缓存。
func VerifyEmailCaptcha(email, userInput string) (bool, error) {
	key := redis_cache.GenerateCaptcha(email)

	storedCaptcha, err := redis_cache.Rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redisCli.Nil {
			return false, nil
		}
		return false, err
	}

	if strings.EqualFold(storedCaptcha, userInput) {
		_ = redis_cache.Rdb.Del(ctx, key).Err()
		return true, nil
	}

	return false, nil
}
