package tts

import (
	"GopherAI/common/code"
	"GopherAI/common/tts"
	"GopherAI/controller"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TTSRequest struct {
	Text string `json:"text,omitempty"`
}

type TTSServices struct {
	ttsService *tts.TTSService
}

func NewTTSServices() *TTSServices {
	return &TTSServices{
		ttsService: tts.NewTTSService(),
	}
}

func CreateTTSTask(c *gin.Context) {
	ttsSvc := NewTTSServices()
	req := new(TTSRequest)
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	if req.Text == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	// 创建TTS任务并返回任务ID，由前端轮询查询结果
	taskID, err := ttsSvc.ttsService.CreateTTS(c, req.Text)
	if err != nil {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.TTSFail,
			Msg:  code.TTSFail.Msg(),
		})
		return
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: []interface{}{gin.H{"task_id": taskID}},
	})
}

func QueryTTSTask(c *gin.Context) {
	ttsSvc := NewTTSServices()
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.CodeInvalidParams,
			Msg:  code.CodeInvalidParams.Msg(),
		})
		return
	}

	TTSQueryResponse, err := ttsSvc.ttsService.QueryTTSFull(c, taskID)
	if err != nil {
		log.Println("语音合成失败", err.Error())
		c.JSON(http.StatusOK, controller.Response{
			Code: code.TTSFail,
			Msg:  code.TTSFail.Msg(),
		})
		return
	}

	if len(TTSQueryResponse.TasksInfo) == 0 {
		c.JSON(http.StatusOK, controller.Response{
			Code: code.TTSFail,
			Msg:  code.TTSFail.Msg(),
		})
		return
	}

	taskInfo := TTSQueryResponse.TasksInfo[0]
	result := gin.H{
		"task_id":     taskInfo.TaskID,
		"task_status": taskInfo.TaskStatus,
	}

	// 检查 TaskResult 是否为 nil，避免空指针异常
	if taskInfo.TaskResult != nil {
		result["task_result"] = taskInfo.TaskResult.SpeechURL
	}

	c.JSON(http.StatusOK, controller.Response{
		Code: code.CodeSuccess,
		Msg:  code.CodeSuccess.Msg(),
		Data: []interface{}{result},
	})
}