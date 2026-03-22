package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const deepToolFailureGuidance = "这次工具调用没有成功，你可以根据错误修正参数、改用其他工具，或在无法继续调用时直接继续完成任务。"

func deepAgentHandlers() []adk.ChatModelAgentMiddleware {
	return []adk.ChatModelAgentMiddleware{
		newDeepToolErrorMiddleware(),
	}
}

func newDeepToolErrorMiddleware() adk.ChatModelAgentMiddleware {
	return &deepToolErrorMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}
}

type deepToolErrorMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *deepToolErrorMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(callCtx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		result, err := endpoint(callCtx, argumentsInJSON, opts...)
		if shouldPropagateDeepToolError(callCtx, err) {
			return result, err
		}
		if err != nil {
			return formatDeepToolFailureResult(tCtx, err), nil
		}
		return result, nil
	}, nil
}

func (m *deepToolErrorMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	return func(callCtx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		result, err := endpoint(callCtx, argumentsInJSON, opts...)
		if shouldPropagateDeepToolError(callCtx, err) {
			return result, err
		}
		if err != nil {
			return schema.StreamReaderFromArray([]string{formatDeepToolFailureResult(tCtx, err)}), nil
		}
		if result == nil {
			return schema.StreamReaderFromArray([]string{formatDeepToolFailureResult(tCtx, errors.New("tool returned nil stream"))}), nil
		}

		reader, writer := schema.Pipe[string](8)
		go func() {
			defer result.Close()
			defer writer.Close()

			hasChunks := false
			lastEndedWithNewline := false
			for {
				chunk, recvErr := result.Recv()
				if errors.Is(recvErr, io.EOF) {
					return
				}
				if shouldPropagateDeepToolError(callCtx, recvErr) {
					writer.Send("", recvErr)
					return
				}
				if recvErr != nil {
					failure := formatDeepToolFailureResult(tCtx, recvErr)
					if hasChunks && !lastEndedWithNewline {
						failure = "\n" + failure
					}
					writer.Send(failure, nil)
					return
				}

				hasChunks = true
				lastEndedWithNewline = strings.HasSuffix(chunk, "\n")
				writer.Send(chunk, nil)
			}
		}()

		return reader, nil
	}, nil
}

func formatDeepToolFailureResult(tCtx *adk.ToolContext, err error) string {
	toolName := "unknown_tool"
	if tCtx != nil && strings.TrimSpace(tCtx.Name) != "" {
		toolName = strings.TrimSpace(tCtx.Name)
	}

	reason := "未知错误"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		reason = strings.TrimSpace(err.Error())
	}

	return fmt.Sprintf("工具 `%s` 调用失败：%s\n%s", toolName, reason, deepToolFailureGuidance)
}

func shouldPropagateDeepToolError(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	if _, ok := compose.IsInterruptRerunError(err); ok {
		return true
	}
	if ctx != nil {
		if ctxErr := ctx.Err(); errors.Is(ctxErr, context.Canceled) || errors.Is(ctxErr, context.DeadlineExceeded) {
			return true
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "context canceled") || strings.Contains(errText, "context deadline exceeded")
}
