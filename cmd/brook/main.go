// Brook：基于 CloudWeGo Eino 的可配置 Agent 入口。
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"

	"brook/internal/brookdir"
	"brook/internal/launcher"
	"brook/pkg/a2ui"
)

func main() {
	cfgPath := flag.String("config", "", "agent 配置文件路径，默认 ~/.brook/agent.yaml")
	query := flag.String("query", "", "用户输入（非空则非交互运行一次）")
	cpID := flag.String("checkpoint-id", "", "中断恢复用的 checkpoint id")
	resumeInput := flag.String("resume-input", "", "Resume 时写入 session 的 resume_user 字段")
	a2uiOut := flag.Bool("a2ui", false, "将事件以 A2UI JSON Lines 输出到 stdout")
	flag.Parse()

	ctx := context.Background()
	path := *cfgPath
	if path == "" {
		var err error
		path, err = brookdir.Ensure()
		if err != nil {
			slog.Error("brookdir", "err", err)
			os.Exit(1)
		}
	}
	rt, err := launcher.Load(ctx, path)
	if err != nil {
		slog.Error("load", "err", err)
		os.Exit(1)
	}
	logPath, err := brookdir.LogFile()
	if err != nil {
		slog.Error("log path", "err", err)
		os.Exit(1)
	}
	if err := launcher.ApplyObservability(rt.Root, logPath, false); err != nil {
		slog.Error("logging", "err", err)
		os.Exit(1)
	}
	root := rt.Root
	r := rt.Runner
	sessKV := rt.Session

	userText := strings.TrimSpace(*query)
	if userText == "" {
		userText = strings.TrimSpace(root.Agent.UserPrompt)
	}
	if userText == "" {
		userText = "你好，简单介绍一下你能做什么。"
	}

	var iter *adk.AsyncIterator[*adk.AgentEvent]
	var snapSession func() map[string]any
	if *cpID != "" && *resumeInput != "" {
		sessKV["resume_user"] = *resumeInput
		cb, snap := launcher.SessionValuesSyncHandler()
		snapSession = snap
		iter, err = r.Resume(ctx, *cpID, adk.WithSessionValues(sessKV), adk.WithCallbacks(cb))
		if err != nil {
			slog.Error("resume", "err", err)
			os.Exit(1)
		}
	} else {
		cb, snap := launcher.SessionValuesSyncHandler()
		snapSession = snap
		opts := []adk.AgentRunOption{adk.WithSessionValues(sessKV), adk.WithCallbacks(cb)}
		if *cpID != "" {
			opts = append(opts, adk.WithCheckPointID(*cpID))
		}
		iter = r.Query(ctx, userText, opts...)
	}

	if *a2uiOut || root.A2UI.Enabled {
		ver := root.A2UI.Version
		if ver == "" {
			ver = "0.8"
		}
		if err := a2ui.WriteAgentEvents(os.Stdout, iter, ver); err != nil {
			slog.Error("a2ui", "err", err)
			os.Exit(1)
		}
	} else {
		for {
			ev, ok := iter.Next()
			if !ok {
				break
			}
			if ev == nil {
				continue
			}
			if ev.Err != nil {
				slog.Error("agent event error", "err", ev.Err)
				continue
			}
			if ev.Output != nil && ev.Output.MessageOutput != nil {
				printMessage(ev.Output.MessageOutput)
			}
			if ev.Action != nil && ev.Action.Interrupted != nil {
				fmt.Fprintf(os.Stderr, "[interrupt] %#v\n", ev.Action.Interrupted.Data)
			}
		}
	}

	if snapSession != nil {
		launcher.MergeSessionValues(sessKV, snapSession())
	}
	if err := rt.SaveSession(); err != nil {
		slog.Error("save session", "err", err)
		os.Exit(1)
	}
}

func printMessage(mv *adk.MessageVariant) {
	if mv == nil {
		return
	}
	if mv.IsStreaming && mv.MessageStream != nil {
		for {
			msg, err := mv.MessageStream.Recv()
			if err != nil {
				break
			}
			fmt.Print(msg.Content)
		}
		fmt.Println()
		return
	}
	if mv.Message != nil {
		fmt.Println(mv.Message.Content)
	}
}
