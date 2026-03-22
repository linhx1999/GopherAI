package agent

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func TestDeepAgentHandlersIncludesToolErrorMiddleware(t *testing.T) {
	handlers := deepAgentHandlers()
	if len(handlers) != 1 {
		t.Fatalf("unexpected handler count: %d", len(handlers))
	}
	if _, ok := handlers[0].(*deepToolErrorMiddleware); !ok {
		t.Fatalf("unexpected handler type: %T", handlers[0])
	}
}

func TestDeepToolErrorMiddlewareWrapInvokableToolCallReturnsFailureMessage(t *testing.T) {
	middleware := newDeepToolErrorMiddleware().(*deepToolErrorMiddleware)
	wrapped, err := middleware.WrapInvokableToolCall(
		context.Background(),
		func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
			return "", errors.New("file not found")
		},
		&adk.ToolContext{Name: "read_file", CallID: "call_1"},
	)
	if err != nil {
		t.Fatalf("WrapInvokableToolCall returned error: %v", err)
	}

	result, runErr := wrapped(context.Background(), `{}`)
	if runErr != nil {
		t.Fatalf("wrapped invokable tool returned error: %v", runErr)
	}
	if !strings.Contains(result, "工具 `read_file` 调用失败") {
		t.Fatalf("unexpected failure result: %q", result)
	}
	if !strings.Contains(result, deepToolFailureGuidance) {
		t.Fatalf("missing guidance in failure result: %q", result)
	}
}

func TestDeepToolErrorMiddlewareWrapInvokableToolCallPropagatesCancellation(t *testing.T) {
	middleware := newDeepToolErrorMiddleware().(*deepToolErrorMiddleware)
	wrapped, err := middleware.WrapInvokableToolCall(
		context.Background(),
		func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
			return "", context.Canceled
		},
		&adk.ToolContext{Name: "read_file", CallID: "call_1"},
	)
	if err != nil {
		t.Fatalf("WrapInvokableToolCall returned error: %v", err)
	}

	_, runErr := wrapped(context.Background(), `{}`)
	if !errors.Is(runErr, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", runErr)
	}
}

func TestDeepToolErrorMiddlewareWrapStreamableToolCallConvertsInitialError(t *testing.T) {
	middleware := newDeepToolErrorMiddleware().(*deepToolErrorMiddleware)
	wrapped, err := middleware.WrapStreamableToolCall(
		context.Background(),
		func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
			return nil, errors.New("permission denied")
		},
		&adk.ToolContext{Name: "write_file", CallID: "call_2"},
	)
	if err != nil {
		t.Fatalf("WrapStreamableToolCall returned error: %v", err)
	}

	stream, runErr := wrapped(context.Background(), `{}`)
	if runErr != nil {
		t.Fatalf("wrapped streamable tool returned error: %v", runErr)
	}
	result := collectStringStream(t, stream)
	if !strings.Contains(result, "工具 `write_file` 调用失败") {
		t.Fatalf("unexpected stream failure result: %q", result)
	}
}

func TestDeepToolErrorMiddlewareWrapStreamableToolCallConvertsRecvError(t *testing.T) {
	middleware := newDeepToolErrorMiddleware().(*deepToolErrorMiddleware)
	wrapped, err := middleware.WrapStreamableToolCall(
		context.Background(),
		func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
			reader, writer := schema.Pipe[string](2)
			go func() {
				defer writer.Close()
				writer.Send("partial output", nil)
				writer.Send("", errors.New("shell exited unexpectedly"))
			}()
			return reader, nil
		},
		&adk.ToolContext{Name: "execute", CallID: "call_3"},
	)
	if err != nil {
		t.Fatalf("WrapStreamableToolCall returned error: %v", err)
	}

	stream, runErr := wrapped(context.Background(), `{}`)
	if runErr != nil {
		t.Fatalf("wrapped streamable tool returned error: %v", runErr)
	}
	result := collectStringStream(t, stream)
	if !strings.Contains(result, "partial output") {
		t.Fatalf("expected partial output to be preserved, got %q", result)
	}
	if !strings.Contains(result, "工具 `execute` 调用失败") {
		t.Fatalf("expected failure message in stream result, got %q", result)
	}
}

func TestDeepToolErrorMiddlewareWrapStreamableToolCallPropagatesCancellation(t *testing.T) {
	middleware := newDeepToolErrorMiddleware().(*deepToolErrorMiddleware)
	wrapped, err := middleware.WrapStreamableToolCall(
		context.Background(),
		func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
			reader, writer := schema.Pipe[string](1)
			go func() {
				defer writer.Close()
				writer.Send("", context.Canceled)
			}()
			return reader, nil
		},
		&adk.ToolContext{Name: "execute", CallID: "call_4"},
	)
	if err != nil {
		t.Fatalf("WrapStreamableToolCall returned error: %v", err)
	}

	stream, runErr := wrapped(context.Background(), `{}`)
	if runErr != nil {
		t.Fatalf("wrapped streamable tool returned error: %v", runErr)
	}
	defer stream.Close()

	_, recvErr := stream.Recv()
	if !errors.Is(recvErr, context.Canceled) {
		t.Fatalf("expected cancellation error, got %v", recvErr)
	}
}

func collectStringStream(t *testing.T, stream *schema.StreamReader[string]) string {
	t.Helper()
	defer stream.Close()

	var builder strings.Builder
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return builder.String()
		}
		if err != nil {
			t.Fatalf("stream recv error: %v", err)
		}
		builder.WriteString(chunk)
	}
}
