// Package extmw 根据配置名称解析 ChatModelAgentMiddleware（可扩展注册表）。
package extmw

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"

	"github.com/hippowc/brook/pkg/agentconfig"
)

// FromRefs 将配置中的 middleware 名称转为实例；未知名称返回错误。
func FromRefs(ctx context.Context, refs []agentconfig.MiddlewareRef) ([]adk.ChatModelAgentMiddleware, error) {
	var out []adk.ChatModelAgentMiddleware
	for _, r := range refs {
		switch r.Name {
		case "", "noop":
			continue
		default:
			return nil, fmt.Errorf("extmw: unknown middleware %q", r.Name)
		}
	}
	_ = ctx
	return out, nil
}
