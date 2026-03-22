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

func CreateSession(c *gin.Context) {
	userID := c.GetUint("userID")

	createdSession, code_ := sessionService.CreateSession(userID)
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
		Data: gin.H{"session": createdSession},
	})
}

func GetUserSessionsByUserName(c *gin.Context) {
	userID := c.GetUint("userID")

	userSessions, err := sessionService.GetUserSessionsByUserRefID(userID)
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
	userID := c.GetUint("userID")
	sessionID := c.Param("session_id")

	if sessionID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	code_ := sessionService.DeleteSession(userID, sessionID)
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
	userID := c.GetUint("userID")
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

	code_ := sessionService.UpdateSessionTitle(userID, sessionID, req.Title)
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
