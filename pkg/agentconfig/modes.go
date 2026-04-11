package agentconfig

// TabCompletableModes 返回 /agent mode 可补全的模式名（与 AgentMode 一致）。
func TabCompletableModes() []string {
	return []string{
		string(ModeReAct),
		string(ModeDeep),
		string(ModeSequential),
		string(ModeParallel),
		string(ModeLoop),
		string(ModeSupervisor),
		string(ModePlanExecute),
		string(ModeCustom),
	}
}
