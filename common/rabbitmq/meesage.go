package rabbitmq

import (
	"GopherAI/dao/message"
	"GopherAI/model"
	"encoding/json"

	"github.com/streadway/amqp"
)

type MessageMQParam struct {
	SessionID string          `json:"session_id"`
	Content   string          `json:"content"`
	UserName  string          `json:"user_name"`
	Role      string          `json:"role"` // user/assistant
	Index     *int            `json:"index,omitempty"`
	ToolCalls json.RawMessage `json:"tool_calls,omitempty"`
}

func GenerateMessageMQParam(msg *model.Message) ([]byte, error) {
	param := MessageMQParam{
		SessionID: msg.SessionID,
		Content:   msg.Content,
		UserName:  msg.UserName,
		Role:      msg.Role,
		ToolCalls: msg.ToolCalls,
	}
	param.Index = &msg.Index
	return json.Marshal(param)
}

func buildMessageFromParam(param MessageMQParam) (*model.Message, bool) {
	newMsg := &model.Message{
		SessionID: param.SessionID,
		Content:   param.Content,
		UserName:  param.UserName,
		Role:      param.Role,
		ToolCalls: param.ToolCalls,
	}
	if param.Index == nil {
		return newMsg, false
	}
	newMsg.Index = *param.Index
	return newMsg, true
}

func MQMessage(msg *amqp.Delivery) error {
	var param MessageMQParam
	err := json.Unmarshal(msg.Body, &param)
	if err != nil {
		return err
	}

	newMsg, hasIndex := buildMessageFromParam(param)
	if hasIndex {
		_, err = message.CreateMessage(newMsg)
		return err
	}

	return message.CreateMessageWithIndex(newMsg)
}
