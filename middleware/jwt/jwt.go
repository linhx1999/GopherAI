package jwt

import (
	"GopherAI/common/code"
	"GopherAI/common/postgres"
	"GopherAI/controller"
	"GopherAI/utils/myjwt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 读取jwt
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		res := new(controller.Response)

		var token string
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// 兼容 URL 参数传 token
			token = c.Query("token")
		}

		if token == "" {
			c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidToken))
			c.Abort()
			return
		}

		log.Println("token is ", token)
		claims, ok := myjwt.ParseToken(token)
		if !ok {
			c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidToken))
			c.Abort()
			return
		}

		user, err := postgres.GetUserByUserID(claims.UserID)
		if err != nil || user == nil {
			c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidToken))
			c.Abort()
			return
		}

		c.Set("userID", user.ID)
		c.Set("userName", user.Username)
		c.Next()
	}
}
