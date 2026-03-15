package node

import (
	"context"

	"brook-agent/internal/core/memory"
	"brook-agent/internal/model"
)

// Input 描述节点执行入参。
type Input struct {
	Request     *model.AgentRequest
	Session     *memory.Session
	LastMessage *model.Message
}

// Output 描述节点执行结果，统一回流给 hub 进行下一步决策。
type Output struct {
	NextNode string
	Message  *model.Message
	Final    *model.AgentResponse
}

// Node 定义 Agent 执行节点统一接口。
type Node interface {
	Name() string
	Execute(ctx context.Context, in Input) (Output, error)
}
