package user

import (
	"GopherAI/common/code"
	"GopherAI/controller"
	userService "GopherAI/service/user"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	//这里的Username只能是账号登录，和我做的另一个项目有区别（邮箱账号均可)
	LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	//验证码由后端生成，存放到redis中，固然需要先发送一次请求CaptchaRequest,然后用返回的验证码
	//邮箱以及密码进行注册，后续再将账号进行返回
	RegisterRequest struct {
		Email    string `json:"email" binding:"required"`
		Captcha  string `json:"captcha"`
		Password string `json:"password"`
	}

	CaptchaRequest struct {
		Email string `json:"email" binding:"required"`
	}
)

func Login(c *gin.Context) {
	req := new(LoginRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	token, code_ := userService.Login(req.Username, req.Password)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, controller.Response{
			Code: code_,
			Msg:  code_.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: []interface{}{gin.H{"token": token}},
	})
}

func Register(c *gin.Context) {
	req := new(RegisterRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	token, code_ := userService.Register(req.Email, req.Password, req.Captcha)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, controller.Response{
			Code: code_,
			Msg:  code_.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: []interface{}{gin.H{"token": token}},
	})
}

func HandleCaptcha(c *gin.Context) {
	req := new(CaptchaRequest)
	//解析参数
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	//给service层进行处理
	code_ := userService.SendCaptcha(req.Email)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, controller.Response{
			Code: code_,
			Msg:  code_.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
	})
}