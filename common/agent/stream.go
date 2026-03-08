package agent

import (
	"io"

	"github.com/cloudwego/eino/schema"

	"GopherAI/model"
)

type StreamEventType string

const (
	StreamEventTypeToolCall       StreamEventType = "tool_call"
	StreamEventTypeReasoningDelta StreamEventType = "reasoning_delta"
	StreamEventTypeReasoningEnd   StreamEventType = "reasoning_end"
	StreamEventTypeContentDelta   StreamEventType = "content_delta"
)

type StreamEvent struct {
	Type     StreamEventType
	Content  string
	ToolCall *model.ToolCall
}

type StreamResult struct {
	Content          string
	ReasoningContent string
	ToolCalls        []model.ToolCall
}

type StreamSink func(StreamEvent) error

// ResultFromMessage 将同步生成结果转换为统一的聚合结果结构。
func ResultFromMessage(msg *schema.Message) *StreamResult {
	result := &StreamResult{}
	if msg == nil {
		return result
	}

	result.Content = msg.Content
	result.ReasoningContent = msg.ReasoningContent
	if len(msg.ToolCalls) == 0 {
		return result
	}

	result.ToolCalls = make([]model.ToolCall, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, convertToolCall(tc))
	}

	return result
}

// ConsumeStream 统一消费底层消息流，并输出聚合结果与结构化事件。
func ConsumeStream(reader *schema.StreamReader[*schema.Message], thinkingMode bool, sink StreamSink) (*StreamResult, error) {
	defer reader.Close()

	result := &StreamResult{}
	reasoningStarted := false
	reasoningClosed := false

	emit := func(event StreamEvent) error {
		if sink == nil {
			return nil
		}
		return sink(event)
	}

	closeReasoning := func() error {
		if !thinkingMode || !reasoningStarted || reasoningClosed {
			return nil
		}
		reasoningClosed = true
		return emit(StreamEvent{Type: StreamEventTypeReasoningEnd})
	}

	for {
		msg, err := reader.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		for _, tc := range msg.ToolCalls {
			toolCall := convertToolCall(tc)
			result.ToolCalls = append(result.ToolCalls, toolCall)
			toolCallCopy := toolCall
			if err := emit(StreamEvent{
				Type:     StreamEventTypeToolCall,
				ToolCall: &toolCallCopy,
			}); err != nil {
				return nil, err
			}
		}

		if msg.ReasoningContent != "" {
			reasoningStarted = true
			result.ReasoningContent += msg.ReasoningContent
			if err := emit(StreamEvent{
				Type:    StreamEventTypeReasoningDelta,
				Content: msg.ReasoningContent,
			}); err != nil {
				return nil, err
			}
		}

		if msg.Content == "" {
			continue
		}

		if err := closeReasoning(); err != nil {
			return nil, err
		}

		result.Content += msg.Content
		if err := emit(StreamEvent{
			Type:    StreamEventTypeContentDelta,
			Content: msg.Content,
		}); err != nil {
			return nil, err
		}
	}

	if err := closeReasoning(); err != nil {
		return nil, err
	}

	return result, nil
}

func convertToolCall(tc schema.ToolCall) model.ToolCall {
	return model.ToolCall{
		ToolID:    tc.ID,
		Function:  tc.Function.Name,
		Arguments: []byte(tc.Function.Arguments),
	}
}
