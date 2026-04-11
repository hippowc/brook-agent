package launcher

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
)

// SessionValuesSyncHandler 注册 ADK 回调：在每次 Agent OnEnd 时抓取当前 run 的 SessionValues 快照。
// Runner 将 WithSessionValues 注入的 map 复制到内部 runSession，运行期写入的键（如 OutputKey）只存在于内部 map，
// 若不通过回调取回，持久化层保存的仍是启动时那份空/旧数据。
//
// 返回的 snapshot 应在一次 Query/Resume 迭代结束后调用，得到最后一次 OnEnd 时的会话 KV 副本。
func SessionValuesSyncHandler() (callbacks.Handler, func() map[string]any) {
	var mu sync.Mutex
	var last map[string]any
	h := callbacks.NewHandlerBuilder().
		OnStartFn(func(ctx context.Context, _ *callbacks.RunInfo, _ callbacks.CallbackInput) context.Context {
			return ctx
		}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
			_ = output
			if info.Component != adk.ComponentOfAgent {
				return ctx
			}
			snap := adk.GetSessionValues(ctx)
			mu.Lock()
			last = snap
			mu.Unlock()
			return ctx
		}).
		Build()
	return h, func() map[string]any {
		mu.Lock()
		defer mu.Unlock()
		if last == nil {
			return nil
		}
		cp := make(map[string]any, len(last))
		for k, v := range last {
			cp[k] = v
		}
		return cp
	}
}

// MergeSessionValues 将 src 合并进 dst（同键覆盖）。dst 一般为 Runtime.Session。
func MergeSessionValues(dst, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}
