package tui

// TUIHelpText 为输入 /help 时展示的简短说明（完整文档见仓库 doc/agent-configuration-guide.md）。
const TUIHelpText = `Brook TUI 内置命令（在输入框输入）：
  /help          本说明
  /config        用编辑器打开当前 agent.yaml
  /new           开始新会话（新对话存档）
  /agent mode X  切换 agent.mode，并写入该模式默认 mode_config（占位名）后写回文件（X 见下表）

agent.mode 一览（切换时会覆盖 mode_config 为内置占位，可按需再编辑 YAML）：
  react          单 Agent ReAct，mode_config 置空
  deep           DeepAgents；mode_config 置空，可自配 deep / sub_agent_names
  sequential     写入占位 sub_agent_names（顺序执行）
  parallel       同上（并行）
  loop           占位 sub_agent_names + loop_max_iterations
  supervisor     占位 supervisor + sub_agent_names
  plan_execute   占位 planner / executor / replanner
  custom         不支持切换（Brook 未接线）

更全说明：若持有源码，见 doc/agent-configuration-guide.md 与 config/agent.example.yaml。
`
