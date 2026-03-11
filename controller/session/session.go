package session

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"GopherAI/common/code"
	"GopherAI/controller"
	sessionService "GopherAI/service/session"
)

type UpdateSessionTitleRequest struct {
	Title string `json:"title" binding:"required"`
}

func GetUserSessionsByUserName(c *gin.Context) {
	userRefID := c.GetUint("userRefID")

	userSessions, err := sessionService.GetUserSessionsByUserRefID(userRefID)
	if err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeServerBusy,
			Msg:  code.CodeServerBusy.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: gin.H{"sessions": userSessions},
	})
}

func DeleteSession(c *gin.Context) {
	userRefID := c.GetUint("userRefID")
	sessionID := c.Param("session_id")

	if sessionID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	code_ := sessionService.DeleteSession(userRefID, sessionID)
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

func UpdateSessionTitle(c *gin.Context) {
	userRefID := c.GetUint("userRefID")
	sessionID := c.Param("session_id")

	if sessionID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	req := new(UpdateSessionTitleRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	code_ := sessionService.UpdateSessionTitle(userRefID, sessionID, req.Title)
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
