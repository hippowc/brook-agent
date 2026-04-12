// Brook-gateway：基于与 brook 相同的 agent 配置，提供 HTTP 接入外部消息。
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hippowc/brook/internal/brookdir"
	"github.com/hippowc/brook/internal/gateway"
	"github.com/hippowc/brook/internal/launcher"
)

func main() {
	cfgPath := flag.String("config", "", "agent 配置文件路径，默认 ~/.brook/agent.yaml")
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
	if !rt.Root.Gateway.Enabled {
		slog.Error("gateway disabled: set gateway.enabled: true in agent config")
		os.Exit(1)
	}

	store, err := gateway.NewSessionStore(&rt.Root.Gateway)
	if err != nil {
		slog.Error("session store", "err", err)
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

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := gateway.Run(runCtx, rt, store); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(0)
		}
		slog.Error("gateway", "err", err)
		os.Exit(1)
	}
}
