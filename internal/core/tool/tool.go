package tool

import "context"

// Call 是工具执行入参。
type Call struct {
	Name string
	Args map[string]string
}

// Result 是工具执行结果。
type Result struct {
	Output  string
	IsError bool
}

// Tool 定义所有工具实现的统一接口。
type Tool interface {
	Name() string
	Execute(ctx context.Context, call Call) (Result, error)
}
