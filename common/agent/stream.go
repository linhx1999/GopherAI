package agent

import (
	"context"
	"io"

	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type MessageSink func(*schema.Message) error

// CollectAgentMessages 执行同步生成，并返回 Agent 过程中的完整产出消息。
func CollectAgentMessages(ctx context.Context, runner *react.Agent, input []*schema.Message) ([]*schema.Message, *schema.Message, error) {
	opt, future := react.WithMessageFuture()

	resp, err := runner.Generate(ctx, input, opt)
	if err != nil {
		return nil, nil, err
	}

	iter := future.GetMessages()
	produced := make([]*schema.Message, 0, 4)
	for {
		msg, ok, err := iter.Next()
		if err != nil {
			return nil, nil, err
		}
		if !ok {
			break
		}
		produced = append(produced, msg)
	}

	return produced, resp, nil
}

// StreamAgentMessages 执行流式生成，并按原始 chunk 顺序输出 schema.Message。
func StreamAgentMessages(ctx context.Context, runner *react.Agent, input []*schema.Message, sink MessageSink) ([]*schema.Message, error) {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	opt, future := react.WithMessageFuture()
	runErrCh := make(chan error, 1)
	go func() {
		sr, err := runner.Stream(runCtx, input, opt)
		if sr != nil {
			sr.Close()
		}
		runErrCh <- err
	}()

	iter := future.GetMessageStreams()
	produced := make([]*schema.Message, 0, 4)
	for {
		sr, ok, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}

		full, err := collectStreamMessage(sr, sink)
		if err != nil {
			return nil, err
		}
		if full != nil {
			produced = append(produced, full)
		}
	}

	if err := <-runErrCh; err != nil {
		return nil, err
	}
	return produced, nil
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
