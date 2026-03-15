package entry

import (
	"context"

	"brook-agent/internal/frame"
)

// Config 表示 entry 的通用配置。
type Config struct {
	Name string
	Addr string
}

// Entry 定义入口层接口，不同接入协议都需实现该接口。
type Entry interface {
	Name() string
	Start(ctx context.Context, handler frame.Handler) error
}

// Factory 是 entry 构造函数。
type Factory func(cfg Config) Entry
