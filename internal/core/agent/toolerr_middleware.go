package agent

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// toolErrorAsObservationMiddleware 将工具调用失败转为「成功返回的错误文本」，避免 compose.ToolsNode
// 把整个 Run 判为失败（见 eino compose/tool_node.go：普通 err 不会生成 ToolMessage）。
// 中断类错误原样返回，以保留 HITL / 恢复语义。
type toolErrorAsObservationMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func newToolErrorAsObservationMiddleware() adk.ChatModelAgentMiddleware {
	return &toolErrorAsObservationMiddleware{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}
}

func (toolErrorAsObservationMiddleware) WrapInvokableToolCall(ctx context.Context, endpoint adk.InvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		out, err := endpoint(ctx, argumentsInJSON, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return "", err
			}
			return fmt.Sprintf("[tool error] %s: %v", tCtx.Name, err), nil
		}
		return out, nil
	}, nil
}

func (toolErrorAsObservationMiddleware) WrapStreamableToolCall(ctx context.Context, endpoint adk.StreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		sr, err := endpoint(ctx, argumentsInJSON, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			msg := fmt.Sprintf("[tool error] %s: %v", tCtx.Name, err)
			return schema.StreamReaderFromArray([]string{msg}), nil
		}
		return sr, nil
	}, nil
}

func (toolErrorAsObservationMiddleware) WrapEnhancedInvokableToolCall(ctx context.Context, endpoint adk.EnhancedInvokableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedInvokableToolCallEndpoint, error) {
	return func(ctx context.Context, toolArgument *schema.ToolArgument, opts ...tool.Option) (*schema.ToolResult, error) {
		out, err := endpoint(ctx, toolArgument, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			return &schema.ToolResult{
				Parts: []schema.ToolOutputPart{{Type: schema.ToolPartTypeText, Text: fmt.Sprintf("[tool error] %s: %v", tCtx.Name, err)}},
			}, nil
		}
		return out, nil
	}, nil
}

func (toolErrorAsObservationMiddleware) WrapEnhancedStreamableToolCall(ctx context.Context, endpoint adk.EnhancedStreamableToolCallEndpoint, tCtx *adk.ToolContext) (adk.EnhancedStreamableToolCallEndpoint, error) {
	return func(ctx context.Context, toolArgument *schema.ToolArgument, opts ...tool.Option) (*schema.StreamReader[*schema.ToolResult], error) {
		sr, err := endpoint(ctx, toolArgument, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			tr := &schema.ToolResult{
				Parts: []schema.ToolOutputPart{{Type: schema.ToolPartTypeText, Text: fmt.Sprintf("[tool error] %s: %v", tCtx.Name, err)}},
			}
			return schema.StreamReaderFromArray([]*schema.ToolResult{tr}), nil
		}
		return sr, nil
	}, nil
}
