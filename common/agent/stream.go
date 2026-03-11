package agent

import (
	"context"
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type MessageSink func(*schema.Message) error

// CollectAgentMessages 执行同步生成，并返回 Agent 过程中的完整产出消息。
func CollectAgentMessages(ctx context.Context, agent adk.Agent, input []*schema.Message) ([]*schema.Message, *schema.Message, error) {
	iter := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent}).Run(ctx, input)
	produced := make([]*schema.Message, 0, 4)
	var finalMessage *schema.Message

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		messages, err := collectAgentEventMessages(event, nil)
		if err != nil {
			return nil, nil, err
		}
		if len(messages) == 0 {
			continue
		}

		produced = append(produced, messages...)
		finalMessage = messages[len(messages)-1]
	}

	return produced, finalMessage, nil
}

// StreamAgentMessages 执行流式生成，并按原始 chunk 顺序输出 schema.Message。
func StreamAgentMessages(ctx context.Context, agent adk.Agent, input []*schema.Message, sink MessageSink) ([]*schema.Message, error) {
	iter := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	}).Run(ctx, input)

	produced := make([]*schema.Message, 0, 4)
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		messages, err := collectAgentEventMessages(event, sink)
		if err != nil {
			return nil, err
		}
		if len(messages) == 0 {
			continue
		}

		produced = append(produced, messages...)
	}
	return produced, nil
}

func collectAgentEventMessages(event *adk.AgentEvent, sink MessageSink) ([]*schema.Message, error) {
	if event == nil {
		return nil, nil
	}
	if event.Err != nil {
		return nil, event.Err
	}
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil
	}

	msgOutput := event.Output.MessageOutput
	if msgOutput.IsStreaming {
		full, err := collectStreamMessage(msgOutput.MessageStream, sink)
		if err != nil {
			return nil, err
		}
		if full == nil {
			return nil, nil
		}
		return []*schema.Message{full}, nil
	}

	msg, err := msgOutput.GetMessage()
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}

	if sink != nil {
		if err := sink(msg); err != nil {
			return nil, err
		}
	}

	return []*schema.Message{msg}, nil
}

func collectStreamMessage(sr *schema.StreamReader[*schema.Message], sink MessageSink) (*schema.Message, error) {
	defer sr.Close()

	chunks := make([]*schema.Message, 0, 8)
	var lastChunk *schema.Message
	for {
		msg, err := sr.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		lastChunk = msg
		chunks = append(chunks, msg)
		if sink != nil {
			if err := sink(msg); err != nil {
				return nil, err
			}
		}
	}

	if len(chunks) == 0 {
		return nil, nil
	}
	if needsSyntheticFinish(lastChunk) {
		synthetic := buildSyntheticFinishChunk(lastChunk)
		chunks = append(chunks, synthetic)
		if sink != nil {
			if err := sink(synthetic); err != nil {
				return nil, err
			}
		}
	}

	return schema.ConcatMessages(chunks)
}

func needsSyntheticFinish(msg *schema.Message) bool {
	if msg == nil {
		return false
	}
	return msg.ResponseMeta == nil || msg.ResponseMeta.FinishReason == ""
}

func buildSyntheticFinishChunk(msg *schema.Message) *schema.Message {
	if msg == nil {
		return &schema.Message{ResponseMeta: &schema.ResponseMeta{FinishReason: "stop"}}
	}

	chunk := &schema.Message{
		Role:       msg.Role,
		Name:       msg.Name,
		ToolCallID: msg.ToolCallID,
		ToolName:   msg.ToolName,
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: "stop",
		},
	}

	if len(msg.ToolCalls) > 0 {
		chunk.ToolCalls = make([]schema.ToolCall, len(msg.ToolCalls))
		for i, tc := range msg.ToolCalls {
			chunk.ToolCalls[i] = schema.ToolCall{
				Index: tc.Index,
				ID:    tc.ID,
				Type:  tc.Type,
				Function: schema.FunctionCall{
					Name: tc.Function.Name,
				},
			}
		}
	}

	return chunk
}
