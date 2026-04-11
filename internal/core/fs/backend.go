// Package fs 根据配置构造 adk/filesystem.Backend（local 或内存），并生成 filesystem 中间件。
package fs

import (
	"context"
	"fmt"
	"strings"

	fsys "github.com/cloudwego/eino/adk/filesystem"
	middlewarefs "github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-ext/adk/backend/local"

	"brook/pkg/agentconfig"
)

// BackendBundle 聚合 Backend 与可选 Shell，以及已构造好的 ChatModelAgentMiddleware。
type BackendBundle struct {
	Backend        fsys.Backend
	Shell          fsys.Shell
	StreamingShell fsys.StreamingShell
	Middleware     adk.ChatModelAgentMiddleware
}

// Build 按 agent.tools.filesystem 构建；未启用时返回 nil。
func Build(ctx context.Context, spec *agentconfig.Root) (*BackendBundle, error) {
	ft := spec.Agent.Tools.Filesystem
	if ft == nil || !ft.Enabled {
		return nil, nil
	}
	switch strings.ToLower(ft.Backend) {
	case "local":
		return buildLocal(ctx, ft)
	case "in_memory", "inmemory":
		return buildInMemory(ctx)
	default:
		return nil, fmt.Errorf("fs: unknown backend %q", ft.Backend)
	}
}

func buildLocal(ctx context.Context, ft *agentconfig.FilesystemToolsSpec) (*BackendBundle, error) {
	cfg := &local.Config{}
	if ft.Local != nil && ft.Local.StrictCommands {
		cfg.ValidateCommand = func(cmd string) error {
			fields := strings.Fields(cmd)
			if len(fields) == 0 {
				return fmt.Errorf("empty command")
			}
			// 与 ReAct 相比，DeepAgents 的 Task/子 Agent 更常做目录遍历、抽样读文件；
			// 若白名单过窄，会出现「command not allowed in strict mode: find」等错误。
			allowed := map[string]bool{
				"ls": true, "cat": true, "grep": true, "pwd": true, "echo": true,
				"find": true, "head": true, "tail": true, "wc": true, "stat": true,
				"file": true, "diff": true, "du": true, "sort": true, "uniq": true,
			}
			if !allowed[fields[0]] {
				return fmt.Errorf("command not allowed in strict mode: %s", fields[0])
			}
			return nil
		}
	}
	backend, err := local.NewBackend(ctx, cfg)
	if err != nil {
		return nil, err
	}
	b := &BackendBundle{Backend: backend}
	if ft.Shell {
		b.Shell = backend
	}
	if ft.StreamingShell {
		b.StreamingShell = backend
	}
	mw, err := middlewarefs.New(ctx, &middlewarefs.MiddlewareConfig{
		Backend:        b.Backend,
		Shell:          b.Shell,
		StreamingShell: b.StreamingShell,
	})
	if err != nil {
		return nil, err
	}
	b.Middleware = mw
	return b, nil
}

func buildInMemory(ctx context.Context) (*BackendBundle, error) {
	mem := fsys.NewInMemoryBackend()
	b := &BackendBundle{Backend: mem}
	mw, err := middlewarefs.New(ctx, &middlewarefs.MiddlewareConfig{
		Backend: b.Backend,
	})
	if err != nil {
		return nil, err
	}
	b.Middleware = mw
	return b, nil
}
