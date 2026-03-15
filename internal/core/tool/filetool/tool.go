package filetool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"brook-agent/internal/core/tool"
)

const Name = "file"

// Tool 提供跨平台文件读写能力。
type Tool struct{}

func init() {
	tool.Register(Name, func() tool.Tool { return &Tool{} })
}

func (t *Tool) Name() string { return Name }

// Execute 支持 read/write/list 三种基础文件操作。
func (t *Tool) Execute(_ context.Context, call tool.Call) (tool.Result, error) {
	op := call.Args["op"]
	path := call.Args["path"]
	switch op {
	case "read":
		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return tool.Result{IsError: true, Output: err.Error()}, err
		}
		return tool.Result{Output: string(data)}, nil
	case "write":
		content := call.Args["content"]
		if err := os.WriteFile(filepath.Clean(path), []byte(content), 0o644); err != nil {
			return tool.Result{IsError: true, Output: err.Error()}, err
		}
		return tool.Result{Output: "ok"}, nil
	case "list":
		items, err := os.ReadDir(filepath.Clean(path))
		if err != nil {
			return tool.Result{IsError: true, Output: err.Error()}, err
		}
		out := ""
		for _, item := range items {
			out += item.Name() + "\n"
		}
		return tool.Result{Output: out}, nil
	default:
		return tool.Result{IsError: true, Output: "unsupported op"}, fmt.Errorf("unsupported file op: %s", op)
	}
}
