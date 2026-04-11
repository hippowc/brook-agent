package launcher

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"brook/pkg/agentconfig"
)

// ApplyObservability 根据配置将日志写入 logFile，并可选让终端仅显示 Error（TUI 下避免刷屏）。
func ApplyObservability(root *agentconfig.Root, logFile string, quietTTY bool) error {
	level := parseSlogLevel(root.Observability.LogLevel)

	if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
		return err
	}
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
		return nil
	}

	fileH := slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})
	var ttyH slog.Handler
	if quietTTY {
		// TUI 使用 alt screen 时，任何 stderr 输出都会叠在界面下方并打乱布局；日志仍写入文件。
		ttyH = slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: level})
	} else {
		ttyH = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	}

	slog.SetDefault(slog.New(&dupLogger{tty: ttyH, file: fileH}))
	return nil
}

func parseSlogLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type dupLogger struct {
	tty  slog.Handler
	file slog.Handler
}

func (d *dupLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return d.file.Enabled(ctx, level) || d.tty.Enabled(ctx, level)
}

func (d *dupLogger) Handle(ctx context.Context, r slog.Record) error {
	if d.file.Enabled(ctx, r.Level) {
		if err := d.file.Handle(ctx, r); err != nil {
			return err
		}
	}
	if d.tty.Enabled(ctx, r.Level) {
		return d.tty.Handle(ctx, r.Clone())
	}
	return nil
}

func (d *dupLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &dupLogger{
		tty:  d.tty.WithAttrs(attrs),
		file: d.file.WithAttrs(attrs),
	}
}

func (d *dupLogger) WithGroup(name string) slog.Handler {
	return &dupLogger{
		tty:  d.tty.WithGroup(name),
		file: d.file.WithGroup(name),
	}
}
