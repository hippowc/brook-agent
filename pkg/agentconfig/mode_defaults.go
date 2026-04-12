package agentconfig

import "gopkg.in/yaml.v3"

// DefaultModeConfig 在通过 TUI/CLI 切换 agent.mode 时使用的占位配置，保证校验通过且可运行；
// 子 Agent 名称为示例，用户应按业务在 agent.yaml 中改名或增删。
func DefaultModeConfig(mode AgentMode) *ModeConfig {
	switch mode {
	case ModeReAct, ModeDeep:
		return nil
	case ModeSequential, ModeParallel:
		return &ModeConfig{
			SubAgentNames: []string{"step-a", "step-b"},
		}
	case ModeLoop:
		return &ModeConfig{
			SubAgentNames:     []string{"step-a", "step-b"},
			LoopMaxIterations: 5,
		}
	case ModeSupervisor:
		return &ModeConfig{
			Supervisor: &SupervisorModeConfig{
				SupervisorAgentName: "supervisor",
			},
			SubAgentNames: []string{"supervisor", "worker1"},
		}
	case ModePlanExecute:
		return &ModeConfig{
			PlanExecute: &PlanExecuteModeConfig{
				PlannerName:   "planner",
				ExecutorName:  "executor",
				ReplannerName: "replanner",
			},
		}
	case ModeCustom:
		return nil
	default:
		return nil
	}
}

// ModeConfigYAMLMap 将 DefaultModeConfig 转为可写入 YAML 根文档的 map；nil 表示写入 mode_config: null。
func ModeConfigYAMLMap(mode AgentMode) (map[string]any, error) {
	mc := DefaultModeConfig(mode)
	if mc == nil {
		return nil, nil
	}
	b, err := yaml.Marshal(mc)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// ModeSwitchUserHint 切换模式后展示给用户的说明（占位名与文档指引）。
func ModeSwitchUserHint(mode AgentMode) string {
	switch mode {
	case ModeSequential, ModeParallel:
		return "本次切换已重写 mode_config。当前为占位 sub_agent_names（step-a、step-b）；请按流水线角色改名或增删。详见 doc/agent-configuration-guide.md。"
	case ModeLoop:
		return "本次切换已重写 mode_config。当前含 sub_agent_names 与 loop_max_iterations 占位；可按需调整。详见 doc/agent-configuration-guide.md。"
	case ModeSupervisor:
		return "本次切换已重写 mode_config。当前为 supervisor + worker1 占位；请按实际角色修改。详见 doc/agent-configuration-guide.md。"
	case ModePlanExecute:
		return "本次切换已重写 mode_config。当前为 planner/executor/replanner 占位；三名须彼此区分。详见 doc/agent-configuration-guide.md。"
	case ModeDeep:
		return "本次切换已清空 mode_config（Deep 可用默认）；可选配置 mode_config.deep 与 sub_agent_names。详见 doc/agent-configuration-guide.md。"
	case ModeReAct:
		return "本次切换已清空 mode_config（ReAct 无需子 Agent）。"
	default:
		return "已更新 mode_config；详见 doc/agent-configuration-guide.md。"
	}
}
