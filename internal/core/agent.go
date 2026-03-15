package core

import (
	"context"
	"fmt"
	"time"

	"brook-agent/internal/common"
	"brook-agent/internal/core/memory"
	"brook-agent/internal/core/node"
	"brook-agent/internal/model"
)

// Agent 定义核心执行接口。
type Agent interface {
	Run(ctx context.Context, req *model.AgentRequest, stream common.StreamWriter) (*model.AgentResponse, error)
}

// Engine 是默认 Agent 实现。
type Engine struct {
	Memory    memory.Store
	Nodes     *node.Manager
	Emitter   common.Emitter
	MaxRounds int
}

// Run 执行核心节点流程：hub -> llm/tool -> hub，直到 hub 给出结束结果。
// 用户消息由 core 入库，assistant/tool 消息由对应节点写入 memory。
func (e *Engine) Run(ctx context.Context, req *model.AgentRequest, stream common.StreamWriter) (*model.AgentResponse, error) {
	if e.MaxRounds <= 0 {
		e.MaxRounds = 10
	}
	if e.Emitter == nil {
		e.Emitter = common.LogEmitter{}
	}
	if stream == nil {
		stream = common.NopStreamWriter{}
	}

	session, err := e.Memory.GetOrCreate(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	if err := e.Memory.SaveMessage(ctx, req.SessionID, model.Message{
		Role:    model.RoleUser,
		Content: req.Input,
	}); err != nil {
		return nil, err
	}
	session, err = e.Memory.GetOrCreate(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}

	traceID := fmt.Sprintf("%s-%d", req.SessionID, time.Now().UnixNano())
	current := "simplehub"
	for i := 0; i < e.MaxRounds; i++ {
		session, err = e.Memory.GetOrCreate(ctx, req.SessionID)
		if err != nil {
			return nil, err
		}

		_ = e.Emitter.Emit(ctx, common.Event{
			TraceID:   traceID,
			Name:      "node.start",
			Timestamp: time.Now(),
			Fields: map[string]string{
				"node": current,
			},
		})

		n, err := e.Nodes.Get(current)
		if err != nil {
			return nil, err
		}
		out, err := n.Execute(ctx, node.Input{
			Request: req,
			Session: session,
		})
		if err != nil {
			return nil, err
		}

		if out.Message != nil {
			_ = stream.WriteChunk(ctx, common.StreamChunk{
				Type: "message",
				Data: out.Message.Content,
			})
		}

		if out.Final != nil {
			out.Final.TraceID = traceID
			_ = stream.Close(ctx)
			return out.Final, nil
		}
		current = out.NextNode
	}

	return nil, fmt.Errorf("max rounds exceeded")
}
