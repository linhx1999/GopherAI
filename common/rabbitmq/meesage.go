package rabbitmq

import (
	"GopherAI/dao/message"
	"GopherAI/model"
	"encoding/json"
	"time"

	"github.com/streadway/amqp"
)

type MessageMQParam struct {
	MessageID    string          `json:"message_id"`
	SessionRefID uint            `json:"session_ref_id"`
	UserRefID    uint            `json:"user_ref_id"`
	Content      string          `json:"content"`
	Role         string          `json:"role"` // user/assistant
	Index        *int            `json:"index,omitempty"`
	Payload      json.RawMessage `json:"payload,omitempty"`
	ToolCalls    json.RawMessage `json:"tool_calls,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

func GenerateMessageMQParam(msg *model.Message) ([]byte, error) {
	param := MessageMQParam{
		MessageID:    msg.MessageID,
		SessionRefID: msg.SessionRefID,
		UserRefID:    msg.UserRefID,
		Content:      msg.Content,
		Role:         msg.Role,
		Payload:      msg.Payload,
		ToolCalls:    msg.ToolCalls,
		CreatedAt:    msg.CreatedAt,
	}
	param.Index = &msg.Index
	return json.Marshal(param)
}

func buildMessageFromParam(param MessageMQParam) (*model.Message, bool) {
	newMsg := &model.Message{
		MessageID:    param.MessageID,
		SessionRefID: param.SessionRefID,
		UserRefID:    param.UserRefID,
		Role:         param.Role,
		Content:      param.Content,
		Payload:      param.Payload,
		ToolCalls:    param.ToolCalls,
	}
	newMsg.CreatedAt = param.CreatedAt
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
