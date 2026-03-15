package frame

import (
	"context"

	"brook-agent/internal/common"
	"brook-agent/internal/core"
	"brook-agent/internal/model"
)

// Handler 定义 frame 对 entry 暴露的统一处理接口。
type Handler interface {
	Handle(ctx context.Context, req *model.AgentRequest, stream common.StreamWriter) (*model.AgentResponse, error)
}

// Engine 负责承接 entry 输入并调用 core Agent。
type Engine struct {
	Agent core.Agent
}

// Handle 执行标准处理流程：接收请求 -> 调用 agent -> 返回结果。
func (e *Engine) Handle(ctx context.Context, req *model.AgentRequest, stream common.StreamWriter) (*model.AgentResponse, error) {
	return e.Agent.Run(ctx, req, stream)
}
