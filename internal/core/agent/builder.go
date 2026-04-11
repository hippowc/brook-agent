// Package agent 根据 agentconfig 构造 adk.Agent（多种模式）。
package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/compose"
	einomodel "github.com/cloudwego/eino/components/model"

	agentfs "brook/internal/core/fs"
	agentmodel "brook/internal/core/model"
	extmw "brook/internal/extension/middleware"
	"brook/pkg/agentconfig"
)

// Build 从根配置构造 Agent；Custom 模式需自行扩展。
func Build(ctx context.Context, root *agentconfig.Root) (adk.Agent, error) {
	if err := root.Validate(); err != nil {
		return nil, err
	}
	cm, err := agentmodel.NewChatModel(ctx, root)
	if err != nil {
		return nil, err
	}
	bundle, err := agentfs.Build(ctx, root)
	if err != nil {
		return nil, err
	}
	extraMW, err := extmw.FromRefs(ctx, root.Agent.Middlewares)
	if err != nil {
		return nil, err
	}

	switch root.Agent.Mode {
	case agentconfig.ModeReAct:
		return buildReact(ctx, root, cm, bundle, extraMW)
	case agentconfig.ModeDeep:
		return buildDeep(ctx, root, cm, bundle, extraMW)
	case agentconfig.ModeSequential:
		return buildSequential(ctx, root, cm, bundle, extraMW)
	case agentconfig.ModeParallel:
		return buildParallel(ctx, root, cm, bundle, extraMW)
	case agentconfig.ModeLoop:
		return buildLoop(ctx, root, cm, bundle, extraMW)
	case agentconfig.ModeSupervisor:
		return buildSupervisor(ctx, root, cm, bundle, extraMW)
	case agentconfig.ModePlanExecute:
		return buildPlanExecute(ctx, root, cm, bundle, extraMW)
	case agentconfig.ModeCustom:
		return nil, fmt.Errorf("agent: mode custom is not wired in brook; extend internal/core/agent")
	default:
		return nil, fmt.Errorf("agent: unknown mode %q", root.Agent.Mode)
	}
}

func chatHandlers(bundle *agentfs.BackendBundle, extra []adk.ChatModelAgentMiddleware) []adk.ChatModelAgentMiddleware {
	var hs []adk.ChatModelAgentMiddleware
	if bundle != nil && bundle.Middleware != nil {
		hs = append(hs, bundle.Middleware)
	}
	hs = append(hs, extra...)
	return hs
}

func buildReact(ctx context.Context, root *agentconfig.Root, cm einomodel.BaseChatModel, bundle *agentfs.BackendBundle, extra []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	cfg := &adk.ChatModelAgentConfig{
		Name:            root.Agent.Name,
		Description:     root.Agent.Description,
		Instruction:     root.Agent.Instruction,
		Model:           cm,
		MaxIterations:   root.Agent.MaxIterations,
		OutputKey:       root.Memory.OutputKey,
		Handlers:        chatHandlers(bundle, extra),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{},
			ReturnDirectly:  root.Agent.Tools.ReturnDirectly,
		},
	}
	return adk.NewChatModelAgent(ctx, cfg)
}

func buildDeep(ctx context.Context, root *agentconfig.Root, cm einomodel.BaseChatModel, bundle *agentfs.BackendBundle, extra []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	var subs []adk.Agent
	if root.Agent.ModeConfig != nil && len(root.Agent.ModeConfig.SubAgentNames) > 0 {
		var err error
		subs, err = buildNamedAgents(ctx, root, cm, bundle, extra, root.Agent.ModeConfig.SubAgentNames)
		if err != nil {
			return nil, err
		}
	}
	dc := &deep.Config{
		Name:          root.Agent.Name,
		Description:   root.Agent.Description,
		ChatModel:     cm,
		Instruction:   root.Agent.Instruction,
		SubAgents:     subs,
		MaxIteration:  root.Agent.MaxIterations,
		OutputKey:     root.Memory.OutputKey,
		Handlers:      extra,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{},
			ReturnDirectly:  root.Agent.Tools.ReturnDirectly,
		},
		ModelRetryConfig: nil,
	}
	if root.Agent.ModeConfig != nil && root.Agent.ModeConfig.Deep != nil {
		dc.WithoutWriteTodos = root.Agent.ModeConfig.Deep.WithoutWriteTodos
		dc.WithoutGeneralSubAgent = root.Agent.ModeConfig.Deep.WithoutGeneralSubAgent
		if root.Agent.ModeConfig.Deep.MaxIteration > 0 {
			dc.MaxIteration = root.Agent.ModeConfig.Deep.MaxIteration
		}
	}
	if bundle != nil {
		dc.Backend = bundle.Backend
		dc.Shell = bundle.Shell
		dc.StreamingShell = bundle.StreamingShell
	}
	return deep.New(ctx, dc)
}

func buildNamedAgents(ctx context.Context, root *agentconfig.Root, cm einomodel.BaseChatModel, bundle *agentfs.BackendBundle, extra []adk.ChatModelAgentMiddleware, names []string) ([]adk.Agent, error) {
	var out []adk.Agent
	for _, n := range names {
		name := strings.TrimSpace(n)
		if name == "" {
			continue
		}
		cfg := &adk.ChatModelAgentConfig{
			Name:          name,
			Description:   root.Agent.Description + " / " + name,
			Instruction:   root.Agent.Instruction + fmt.Sprintf("\n\n[Your role: sub-agent %q]", name),
			Model:         cm,
			MaxIterations: root.Agent.MaxIterations,
			Handlers:      chatHandlers(bundle, nil),
			ToolsConfig: adk.ToolsConfig{
				ToolsNodeConfig: compose.ToolsNodeConfig{},
				ReturnDirectly:  root.Agent.Tools.ReturnDirectly,
			},
		}
		a, err := adk.NewChatModelAgent(ctx, cfg)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func buildSequential(ctx context.Context, root *agentconfig.Root, cm einomodel.BaseChatModel, bundle *agentfs.BackendBundle, extra []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	names := root.Agent.ModeConfig.SubAgentNames
	subs, err := buildNamedAgents(ctx, root, cm, bundle, extra, names)
	if err != nil {
		return nil, err
	}
	return adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        root.Agent.Name,
		Description: root.Agent.Description,
		SubAgents:   subs,
	})
}

func buildParallel(ctx context.Context, root *agentconfig.Root, cm einomodel.BaseChatModel, bundle *agentfs.BackendBundle, extra []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	subs, err := buildNamedAgents(ctx, root, cm, bundle, extra, root.Agent.ModeConfig.SubAgentNames)
	if err != nil {
		return nil, err
	}
	return adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        root.Agent.Name,
		Description: root.Agent.Description,
		SubAgents:   subs,
	})
}

func buildLoop(ctx context.Context, root *agentconfig.Root, cm einomodel.BaseChatModel, bundle *agentfs.BackendBundle, extra []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	subs, err := buildNamedAgents(ctx, root, cm, bundle, extra, root.Agent.ModeConfig.SubAgentNames)
	if err != nil {
		return nil, err
	}
	maxIter := 3
	if root.Agent.ModeConfig != nil && root.Agent.ModeConfig.LoopMaxIterations > 0 {
		maxIter = root.Agent.ModeConfig.LoopMaxIterations
	}
	return adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          root.Agent.Name,
		Description:   root.Agent.Description,
		SubAgents:     subs,
		MaxIterations: maxIter,
	})
}

func buildSupervisor(ctx context.Context, root *agentconfig.Root, cm einomodel.BaseChatModel, bundle *agentfs.BackendBundle, extra []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	supName := root.Agent.ModeConfig.Supervisor.SupervisorAgentName
	var workers []string
	for _, n := range root.Agent.ModeConfig.SubAgentNames {
		if strings.TrimSpace(n) != "" && n != supName {
			workers = append(workers, n)
		}
	}
	subs, err := buildNamedAgents(ctx, root, cm, bundle, extra, workers)
	if err != nil {
		return nil, err
	}
	sup, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:          supName,
		Description:   root.Agent.Description + " (supervisor)",
		Instruction:   root.Agent.Instruction + "\n\nYou coordinate sub-agents.",
		Model:         cm,
		MaxIterations: root.Agent.MaxIterations,
		Handlers:      chatHandlers(bundle, extra),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{},
			ReturnDirectly:    root.Agent.Tools.ReturnDirectly,
		},
	})
	if err != nil {
		return nil, err
	}
	return supervisor.New(ctx, &supervisor.Config{
		Supervisor: sup,
		SubAgents:  subs,
	})
}

func buildPlanExecute(ctx context.Context, root *agentconfig.Root, cm einomodel.BaseChatModel, bundle *agentfs.BackendBundle, extra []adk.ChatModelAgentMiddleware) (adk.Agent, error) {
	pe := root.Agent.ModeConfig.PlanExecute
	names := []string{pe.PlannerName, pe.ExecutorName, pe.ReplannerName}
	agents, err := buildNamedAgents(ctx, root, cm, bundle, extra, names)
	if err != nil {
		return nil, err
	}
	byName := map[string]adk.Agent{}
	for _, a := range agents {
		byName[a.Name(ctx)] = a
	}
	planner := byName[pe.PlannerName]
	exec := byName[pe.ExecutorName]
	replan := byName[pe.ReplannerName]
	if planner == nil || exec == nil || replan == nil {
		return nil, fmt.Errorf("agent: plan_execute agents not found for names %+v", pe)
	}
	return planexecute.New(ctx, &planexecute.Config{
		Planner:       planner,
		Executor:      exec,
		Replanner:     replan,
		MaxIterations: root.Agent.MaxIterations,
	})
}
