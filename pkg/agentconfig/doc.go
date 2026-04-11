// Package agentconfig 提供与 CloudWeGo Eino / ADK 对齐的声明式配置模型。
//
// 映射关系摘要：
//   - AgentSpec.Mode -> ChatModelAgent / deep.New / Sequential|Parallel|Loop / supervisor / planexecute
//   - AgentSpec.Instruction, MaxIterations, Tools -> ChatModelAgentConfig（Instruction/UserPrompt 可为 @文件路径，见 ExpandAtFileRefs）
//   - ModelsSpec -> 构造 eino-ext 各 Provider 的 ToolCallingChatModel
//   - ToolsSpec.Filesystem -> adk/middlewares/filesystem + filesystem.Backend（如 eino-ext local）
//   - MemorySpec.OutputKey -> ChatModelAgentConfig.OutputKey；其余为业务层 Session 持久化
//   - InterruptSpec -> adk.RunnerConfig.CheckPointStore
//
// 加载示例：agentconfig.LoadFile("config/agent.yaml")

package agentconfig
