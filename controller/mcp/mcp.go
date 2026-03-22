package mcp

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"GopherAI/common/code"
	"GopherAI/controller"
	mcpService "GopherAI/service/mcp"
)

type UpsertServerRequest struct {
	Name          string                   `json:"name" binding:"required"`
	TransportType string                   `json:"transport_type"`
	Endpoint      string                   `json:"endpoint" binding:"required"`
	Headers       []mcpService.HeaderInput `json:"headers"`
}

func ListServers(c *gin.Context) {
	servers, code_ := mcpService.ListServers(c.GetUint("userID"))
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
		Data: gin.H{"servers": servers},
	})
}

func GetServer(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	server, code_ := mcpService.GetServerDetail(c.GetUint("userID"), serverID)
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
		Data: gin.H{"server": server},
	})
}

func CreateServer(c *gin.Context) {
	req := new(UpsertServerRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	server, code_ := mcpService.CreateServer(c.GetUint("userID"), mcpService.UpsertServerInput{
		Name:          req.Name,
		TransportType: req.TransportType,
		Endpoint:      req.Endpoint,
		Headers:       req.Headers,
	})
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
		Data: gin.H{"server": server},
	})
}

func UpdateServer(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	req := new(UpsertServerRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	server, code_ := mcpService.UpdateServer(c.GetUint("userID"), serverID, mcpService.UpsertServerInput{
		Name:          req.Name,
		TransportType: req.TransportType,
		Endpoint:      req.Endpoint,
		Headers:       req.Headers,
	})
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
		Data: gin.H{"server": server},
	})
}

func DeleteServer(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	code_ := mcpService.DeleteServer(c.GetUint("userID"), serverID)
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

func TestServer(c *gin.Context) {
	serverID := c.Param("server_id")
	if serverID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	server, code_ := mcpService.TestServer(c.Request.Context(), c.GetUint("userID"), serverID)
	if code_ != code.CodeSuccess {
		c.JSON(http.StatusOK, controller.Response{
			Code: code_,
			Msg:  code_.Msg(),
			Data: gin.H{"server": server},
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: gin.H{"server": server},
	})
}
