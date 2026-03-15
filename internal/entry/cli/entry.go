package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"brook-agent/internal/common"
	"brook-agent/internal/entry"
	"brook-agent/internal/frame"
	"brook-agent/internal/model"
)

const Name = "cli"

type Entry struct {
	cfg entry.Config
}

func init() {
	entry.Register(Name, func(cfg entry.Config) entry.Entry {
		return &Entry{cfg: cfg}
	})
}

func (e *Entry) Name() string { return Name }

// Start 启动命令行入口，读取标准输入并输出结果。
func (e *Entry) Start(ctx context.Context, handler frame.Handler) error {
	reader := bufio.NewReader(os.Stdin)
	sessionID := "cli-session"
	for {
		fmt.Print("input> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.EqualFold(line, "exit") || strings.EqualFold(line, "quit") {
			fmt.Println("bye")
			return nil
		}
		resp, err := handler.Handle(ctx, &model.AgentRequest{
			SessionID: sessionID,
			Input:     line,
		}, common.NopStreamWriter{})
		if err != nil {
			return err
		}
		fmt.Println(resp.Output)
	}
}
