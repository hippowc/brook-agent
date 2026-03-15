package toolnode

import (
	"context"
	"fmt"
	"strings"

	"brook-agent/internal/core/memory"
	"brook-agent/internal/core/node"
	"brook-agent/internal/core/tool"
	"brook-agent/internal/model"
)

const Name = "tool"

// Node 负责执行 assistant 产出的工具调用。
type Node struct {
	tools  *tool.Manager
	memory memory.Store
}

func init() {
	node.Register(Name, func(cfg node.BuildConfig) node.Node {
		return &Node{
			tools:  cfg.Tools,
			memory: cfg.Memory,
		}
	})
}

func (n *Node) Name() string { return Name }

func (n *Node) Execute(ctx context.Context, in node.Input) (node.Output, error) {
	session, err := n.memory.GetOrCreate(ctx, in.Request.SessionID)
	if err != nil {
		return node.Output{}, err
	}
	if len(session.Messages) == 0 {
		return node.Output{}, fmt.Errorf("empty session messages")
	}
	last := session.Messages[len(session.Messages)-1]
	if len(last.ToolCalls) == 0 {
		return node.Output{NextNode: "simplehub"}, nil
	}

	var merged []string
	for _, call := range last.ToolCalls {
		t, err := n.tools.Get(call.Name)
		if err != nil {
			text := fmt.Sprintf("[%s] %v", call.Name, err)
			merged = append(merged, text)
			if saveErr := n.memory.SaveToolResult(ctx, in.Request.SessionID, model.ToolResult{
				CallID:  call.ID,
				Name:    call.Name,
				Output:  text,
				IsError: true,
			}); saveErr != nil {
				return node.Output{}, saveErr
			}
			if saveErr := n.memory.SaveMessage(ctx, in.Request.SessionID, model.Message{
				Role:       model.RoleTool,
				Name:       call.Name,
				ToolCallID: call.ID,
				Content:    text,
			}); saveErr != nil {
				return node.Output{}, saveErr
			}
			continue
		}
		ret, err := t.Execute(ctx, tool.Call{
			Name: call.Name,
			Args: call.Args,
		})
		text := ret.Output
		if err != nil {
			text = fmt.Sprintf("[%s] %v", call.Name, err)
		}
		merged = append(merged, fmt.Sprintf("[%s] %s", call.Name, text))
		if saveErr := n.memory.SaveToolResult(ctx, in.Request.SessionID, model.ToolResult{
			CallID:  call.ID,
			Name:    call.Name,
			Output:  text,
			IsError: err != nil || ret.IsError,
		}); saveErr != nil {
			return node.Output{}, saveErr
		}
		if saveErr := n.memory.SaveMessage(ctx, in.Request.SessionID, model.Message{
			Role:       model.RoleTool,
			Name:       call.Name,
			ToolCallID: call.ID,
			Content:    text,
		}); saveErr != nil {
			return node.Output{}, saveErr
		}
	}

	msg := model.Message{
		Role:    model.RoleTool,
		Name:    "tool-node",
		Content: strings.Join(merged, "\n"),
	}
	return node.Output{
		NextNode: "simplehub",
		Message:  &msg,
	}, nil
}
