package bashtool

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"brook-agent/internal/core/tool"
)

const Name = "bash"

// Tool 负责跨平台命令执行。
type Tool struct{}

func init() {
	tool.Register(Name, func() tool.Tool { return &Tool{} })
}

func (t *Tool) Name() string { return Name }

// Execute 根据运行平台选择 shell 执行命令。
func (t *Tool) Execute(ctx context.Context, call tool.Call) (tool.Result, error) {
	cmdText := call.Args["command"]
	if cmdText == "" {
		return tool.Result{IsError: true, Output: "command is required"}, fmt.Errorf("command is required")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", cmdText)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-lc", cmdText)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return tool.Result{IsError: true, Output: out.String()}, err
	}
	return tool.Result{Output: out.String()}, nil
}
