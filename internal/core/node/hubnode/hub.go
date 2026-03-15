package hubnode

import (
	"context"

	"brook-agent/internal/core/node"
	"brook-agent/internal/model"
)

const Name = "hub"

// Node 是中枢节点，负责在 llm/tool/结束 之间做路由决策。
type Node struct{}

func init() {
	node.Register(Name, func(_ node.BuildConfig) node.Node {
		return &Node{}
	})
}

func (n *Node) Name() string { return Name }

func (n *Node) Execute(_ context.Context, in node.Input) (node.Output, error) {
	if len(in.Session.Messages) == 0 {
		return node.Output{
			NextNode: "llm",
		}, nil
	}

	last := in.Session.Messages[len(in.Session.Messages)-1]
	if last.Role == model.RoleAssistant && len(last.ToolCalls) > 0 {
		return node.Output{NextNode: "tool"}, nil
	}

	if last.Role == model.RoleTool {
		return node.Output{NextNode: "llm"}, nil
	}

	if last.Role == model.RoleAssistant {
		return node.Output{
			Final: &model.AgentResponse{
				SessionID: in.Request.SessionID,
				Output:    last.Content,
				Finished:  true,
			},
		}, nil
	}

	return node.Output{NextNode: "llm"}, nil
}
