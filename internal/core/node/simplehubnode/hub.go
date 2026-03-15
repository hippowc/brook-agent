package simplehubnode

import (
	"context"

	"brook-agent/internal/core/node"
	"brook-agent/internal/model"
)

const Name = "simplehub"

// Node 是简化中枢节点，只负责做流程路由与结束判定。
type Node struct{}

func init() {
	node.Register(Name, func(_ node.BuildConfig) node.Node {
		return &Node{}
	})
}

func (n *Node) Name() string { return Name }

func (n *Node) Execute(_ context.Context, in node.Input) (node.Output, error) {
	if len(in.Session.Messages) == 0 {
		return node.Output{NextNode: "llm"}, nil
	}

	last := in.Session.Messages[len(in.Session.Messages)-1]
	switch last.Role {
	case model.RoleAssistant:
		if len(last.ToolCalls) > 0 {
			return node.Output{NextNode: "tool"}, nil
		}
		return node.Output{
			Final: &model.AgentResponse{
				SessionID: in.Request.SessionID,
				Output:    last.Content,
				Finished:  true,
			},
		}, nil
	case model.RoleTool:
		return node.Output{NextNode: "llm"}, nil
	default:
		return node.Output{NextNode: "llm"}, nil
	}
}
