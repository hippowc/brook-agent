// Package extcallbacks 注册全局 callbacks，实现基础可观测性（日志）。
package extcallbacks

import (
	"context"
	"log/slog"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
)

// SetupLogging 注册全局 Handler，输出组件 OnStart/OnEnd/OnError（需进程启动时调用一次）。
func SetupLogging() {
	h := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if info != nil {
				slog.InfoContext(ctx, "eino.callback.on_start", "component", info.Component, "type", info.Type, "name", info.Name)
			}
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			if info == nil {
				return ctx
			}
			if mo := model.ConvCallbackOutput(output); mo != nil && mo.Message != nil &&
				mo.Message.ResponseMeta != nil && mo.Message.ResponseMeta.Usage != nil {
				slog.InfoContext(ctx, "eino.callback.on_end", "component", info.Component, "name", info.Name,
					"tokens", mo.Message.ResponseMeta.Usage.TotalTokens)
			} else {
				slog.InfoContext(ctx, "eino.callback.on_end", "component", info.Component, "name", info.Name)
			}
			return ctx
		}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			slog.ErrorContext(ctx, "eino.callback.on_error", "component", info.Component, "name", info.Name, "err", err)
			return ctx
		}).
		Build()
	callbacks.AppendGlobalHandlers(h)
}
