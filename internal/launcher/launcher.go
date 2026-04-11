// Package launcher 从配置文件构造 adk.Runner 与会话状态，供 CLI / TUI 共用。
package launcher

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"

	agentrun "brook/internal/core/agent"
	extcallbacks "brook/internal/extension/callbacks"
	"brook/internal/business/store"
	"brook/pkg/agentconfig"
)

// Runtime 持有一次运行所需的配置与 Runner。
type Runtime struct {
	Root    *agentconfig.Root
	Runner  *adk.Runner
	Session map[string]any
}

// Load 读取 YAML、构建 Agent、装配 Runner（含可选 checkpoint）。
func Load(ctx context.Context, cfgPath string) (*Runtime, error) {
	root, err := agentconfig.LoadFile(cfgPath)
	if err != nil {
		return nil, err
	}
	if len(root.Observability.GlobalHandlers) == 0 {
		extcallbacks.SetupLogging()
	}

	ag, err := agentrun.Build(ctx, root)
	if err != nil {
		return nil, err
	}

	var sess map[string]any
	if root.Memory.SessionStore == "file" && root.Memory.SessionFilePath != "" {
		sf := store.SessionFile{Path: root.Memory.SessionFilePath}
		sess, err = sf.Load()
		if err != nil {
			return nil, err
		}
	} else {
		sess = map[string]any{}
	}

	rc := adk.RunnerConfig{
		Agent:           ag,
		EnableStreaming: true,
	}
	if root.Interrupt.Enabled {
		switch strings.ToLower(root.Interrupt.CheckpointBackend) {
		case "file":
			st, err := store.NewFileCheckPointStore(root.Interrupt.CheckpointFilePath)
			if err != nil {
				return nil, err
			}
			rc.CheckPointStore = st
		}
	}

	r := adk.NewRunner(ctx, rc)
	return &Runtime{Root: root, Runner: r, Session: sess}, nil
}

// SaveSession 将会话写回文件（若配置了 file store）。
func (rt *Runtime) SaveSession() error {
	root := rt.Root
	if root.Memory.SessionStore != "file" || root.Memory.SessionFilePath == "" {
		return nil
	}
	return (&store.SessionFile{Path: root.Memory.SessionFilePath}).Save(rt.Session)
}

// QuietLogs 关闭向终端输出结构化日志，避免与 TUI 打架。
func QuietLogs() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
}
