package agent

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type fakeAgent struct {
	events    []*adk.AgentEvent
	lastInput *adk.AgentInput
}

func (f *fakeAgent) Name(context.Context) string {
	return "fake"
}

func (f *fakeAgent) Description(context.Context) string {
	return "fake agent"
}

func (f *fakeAgent) Run(_ context.Context, input *adk.AgentInput, _ ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	f.lastInput = input

	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	go func() {
		defer gen.Close()
		for _, event := range f.events {
			gen.Send(event)
		}
	}()

	return iter
}

func newMessageEvent(msg *schema.Message) *adk.AgentEvent {
	return &adk.AgentEvent{
		Output: &adk.AgentOutput{
			MessageOutput: &adk.MessageVariant{
				Message: msg,
				Role:    msg.Role,
			},
		},
	}
}

func TestCollectStreamMessageConcatsChunksAndEmitsSyntheticFinish(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{
			Role:             schema.Assistant,
			ReasoningContent: "先分析",
		},
		{
			Role:    schema.Assistant,
			Content: "答案",
		},
	})

	var emitted []*schema.Message
	full, err := collectStreamMessage(reader, &StreamMessageSink{
		OnChunk: func(msg *schema.Message) error {
			emitted = append(emitted, msg)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("collectStreamMessage returned error: %v", err)
	}
	if full == nil {
		t.Fatal("expected full message")
	}
	if full.Content != "答案" {
		t.Fatalf("unexpected content: %q", full.Content)
	}
	if full.ReasoningContent != "先分析" {
		t.Fatalf("unexpected reasoning content: %q", full.ReasoningContent)
	}
	if full.ResponseMeta == nil || full.ResponseMeta.FinishReason != "stop" {
		t.Fatalf("expected synthetic finish reason, got %#v", full.ResponseMeta)
	}
	if len(emitted) != 3 {
		t.Fatalf("expected 3 emitted chunks, got %d", len(emitted))
	}
	if emitted[2].ResponseMeta == nil || emitted[2].ResponseMeta.FinishReason != "stop" {
		t.Fatalf("expected last emitted chunk to be synthetic finish, got %#v", emitted[2].ResponseMeta)
	}
}

func TestCollectStreamMessageKeepsExistingFinishReason(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{
			Role:    schema.Tool,
			Content: `{"result":"ok"}`,
			ResponseMeta: &schema.ResponseMeta{
				FinishReason: "stop",
			},
		},
	})

	var finishReasons []string
	full, err := collectStreamMessage(reader, &StreamMessageSink{
		OnChunk: func(msg *schema.Message) error {
			if msg.ResponseMeta != nil {
				finishReasons = append(finishReasons, msg.ResponseMeta.FinishReason)
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("collectStreamMessage returned error: %v", err)
	}
	if full == nil || full.ResponseMeta == nil || full.ResponseMeta.FinishReason != "stop" {
		t.Fatalf("unexpected full message response meta: %#v", full)
	}
	if !reflect.DeepEqual(finishReasons, []string{"stop"}) {
		t.Fatalf("unexpected finish reasons: %v", finishReasons)
	}
}

func TestCollectStreamMessageStopsOnSinkError(t *testing.T) {
	reader := schema.StreamReaderFromArray([]*schema.Message{
		{Role: schema.Assistant, Content: "答"},
	})

	expectedErr := errors.New("sink failed")
	_, err := collectStreamMessage(reader, &StreamMessageSink{
		OnChunk: func(msg *schema.Message) error {
			return expectedErr
		},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected sink error, got %v", err)
	}
}

func TestCollectAgentMessagesSkipsNonMessageEvents(t *testing.T) {
	agent := &fakeAgent{
		events: []*adk.AgentEvent{
			{},
			newMessageEvent(&schema.Message{Role: schema.Assistant, Content: "第一条"}),
			newMessageEvent(&schema.Message{Role: schema.Assistant, Content: "最终答案"}),
		},
	}

	produced, finalMessage, err := CollectAgentMessages(context.Background(), agent, []*schema.Message{
		{Role: schema.User, Content: "你好"},
	})
	if err != nil {
		t.Fatalf("CollectAgentMessages returned error: %v", err)
	}
	if agent.lastInput == nil || len(agent.lastInput.Messages) != 1 {
		t.Fatalf("unexpected input passed to agent: %#v", agent.lastInput)
	}
	if len(produced) != 2 {
		t.Fatalf("expected 2 produced messages, got %d", len(produced))
	}
	if finalMessage == nil || finalMessage.Content != "最终答案" {
		t.Fatalf("unexpected final message: %#v", finalMessage)
	}
}

func TestCollectAgentMessagesReturnsEventError(t *testing.T) {
	expectedErr := errors.New("agent failed")
	agent := &fakeAgent{
		events: []*adk.AgentEvent{
			{Err: expectedErr},
		},
	}

	_, _, err := CollectAgentMessages(context.Background(), agent, nil)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected event error, got %v", err)
	}
}

func TestStreamAgentMessagesConcatsStreamingEvent(t *testing.T) {
	agent := &fakeAgent{
		events: []*adk.AgentEvent{
			{
				Output: &adk.AgentOutput{
					MessageOutput: &adk.MessageVariant{
						IsStreaming: true,
						Role:        schema.Assistant,
						MessageStream: schema.StreamReaderFromArray([]*schema.Message{
							{Role: schema.Assistant, ReasoningContent: "先分析"},
							{Role: schema.Assistant, Content: "答案"},
						}),
					},
				},
			},
		},
	}

	var emitted []*schema.Message
	var completed []*schema.Message
	produced, err := StreamAgentMessages(context.Background(), agent, nil, &StreamMessageSink{
		OnChunk: func(msg *schema.Message) error {
			emitted = append(emitted, msg)
			return nil
		},
		OnComplete: func(msg *schema.Message) error {
			completed = append(completed, msg)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("StreamAgentMessages returned error: %v", err)
	}
	if agent.lastInput == nil || !agent.lastInput.EnableStreaming {
		t.Fatalf("expected streaming input, got %#v", agent.lastInput)
	}
	if len(produced) != 1 {
		t.Fatalf("expected 1 produced message, got %d", len(produced))
	}
	if produced[0].Content != "答案" || produced[0].ReasoningContent != "先分析" {
		t.Fatalf("unexpected produced message: %#v", produced[0])
	}
	if len(emitted) != 3 {
		t.Fatalf("expected 3 emitted chunks, got %d", len(emitted))
	}
	if emitted[2].ResponseMeta == nil || emitted[2].ResponseMeta.FinishReason != "stop" {
		t.Fatalf("expected synthetic finish chunk, got %#v", emitted[2])
	}
	if len(completed) != 1 {
		t.Fatalf("expected 1 completed message, got %d", len(completed))
	}
	if completed[0].Content != "答案" || completed[0].ReasoningContent != "先分析" {
		t.Fatalf("unexpected completed message: %#v", completed[0])
	}
}
