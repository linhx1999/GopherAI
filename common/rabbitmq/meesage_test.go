package rabbitmq

import (
	"encoding/json"
	"testing"

	"GopherAI/model"
)

func TestGenerateMessageMQParamPreservesZeroIndexAndToolCalls(t *testing.T) {
	msg := &model.Message{
		MessageID:    "msg-1",
		SessionRefID: 101,
		UserRefID:    202,
		Content:      "hello",
		Role:         "assistant",
		Index:        0,
		Payload:      json.RawMessage(`{"role":"assistant","content":"hello","reasoning_content":"先想"}`),
		ToolCalls:    json.RawMessage(`[{"tool_id":"tool-1","function":"knowledge_search","arguments":{"query":"go"}}]`),
	}

	data, err := GenerateMessageMQParam(msg)
	if err != nil {
		t.Fatalf("GenerateMessageMQParam returned error: %v", err)
	}

	var param MessageMQParam
	if err := json.Unmarshal(data, &param); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if param.Index == nil || *param.Index != 0 {
		t.Fatalf("expected explicit zero index, got %#v", param.Index)
	}
	if string(param.ToolCalls) != string(msg.ToolCalls) {
		t.Fatalf("unexpected tool calls payload: %s", string(param.ToolCalls))
	}
	if string(param.Payload) != string(msg.Payload) {
		t.Fatalf("unexpected payload: %s", string(param.Payload))
	}
}

func TestBuildMessageFromParamBackwardCompatibleWithoutIndex(t *testing.T) {
	var param MessageMQParam
	if err := json.Unmarshal([]byte(`{"message_id":"msg-2","session_ref_id":12,"user_ref_id":34,"content":"hi","role":"user","payload":{"role":"user","content":"hi"}}`), &param); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	msg, hasIndex := buildMessageFromParam(param)
	if hasIndex {
		t.Fatal("expected missing index to use compatibility path")
	}
	if msg.MessageID != "msg-2" || msg.SessionRefID != 12 || msg.UserRefID != 34 || msg.Content != "hi" || msg.Role != "user" {
		t.Fatalf("unexpected message: %#v", msg)
	}
	if string(msg.Payload) != `{"role":"user","content":"hi"}` {
		t.Fatalf("unexpected payload: %s", string(msg.Payload))
	}
}
