// Package agentconfig 定义与 github.com/cloudwego/eino / ADK 概念对齐的可加载配置结构，
// 用于在运行时构造 ChatModelAgent、DeepAgents、Workflow Agents、Runner 等，而不直接依赖具体实现细节。
package agentconfig

// Root 为单文件根配置，对应一次应用或一条业务线的 agent 定义。
type Root struct {
	// Version 配置格式版本，便于后续迁移。
	Version string `yaml:"version" json:"version"`

	// Agent 编排模式与通用行为（对应 adk 中 ChatModelAgent / prebuilt / workflow 等）。
	Agent AgentSpec `yaml:"agent" json:"agent"`

	// Models 多模型与当前选用模型（对应 components/model.ToolCallingChatModel 的构建参数）。
	Models ModelsSpec `yaml:"models" json:"models"`

	// Memory 记忆与持久化侧策略（对应 History / SessionValues / 外部 store 的业务封装）。
	Memory MemorySpec `yaml:"memory,omitempty" json:"memory,omitempty"`

	// Observability 可观测性（对应 callbacks.Handler 注册与 DevOps 集成）。
	Observability ObservabilitySpec `yaml:"observability,omitempty" json:"observability,omitempty"`

	// Interrupt 中断与恢复（对应 adk.Runner + CheckPointStore + ResumableAgent）。
	Interrupt InterruptSpec `yaml:"interrupt,omitempty" json:"interrupt,omitempty"`

	// A2UI 将 Agent 事件流映射为 A2UI 兼容的 JSONL（见 pkg/a2ui）。
	A2UI A2UISpec `yaml:"a2ui,omitempty" json:"a2ui,omitempty"`
}

// A2UISpec 控制是否输出 A2UI 风格的流式 UI 消息（JSON Lines）。
type A2UISpec struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"` // 如 "0.8"，默认 0.8
}

// AgentSpec 描述「跑哪一种 ADK 流程」以及该流程下的 agent 级参数。
type AgentSpec struct {
	// Mode 决定使用哪类 eino ADK 组合方式。
	//   react          — ChatModelAgent（内置 ReAct）
	//   deep           — adk/prebuilt/deep
	//   sequential     — Sequential Agent（子 agent 顺序执行）
	//   parallel       — Parallel Agent
	//   loop           — Loop Agent
	//   supervisor     — prebuilt/supervisor
	//   plan_execute   — prebuilt/planexecute
	//   custom         — 由代码按 Name 解析，不由此文件单独描述图结构
	Mode AgentMode `yaml:"mode" json:"mode"`

	// Name / Description 映射 adk.Agent 的 Name(ctx)、Description(ctx) 语义。
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Instruction 系统提示词，对应 ChatModelAgentConfig.Instruction（支持 Session 占位符时由 GenModelInput 渲染）。
	Instruction string `yaml:"instruction" json:"instruction"`

	// UserPrompt 默认用户侧提示模板或首轮用户消息模板（业务层可渲染后写入 AgentInput.Messages）。
	UserPrompt string `yaml:"user_prompt,omitempty" json:"user_prompt,omitempty"`

	// MaxIterations ReAct 最大轮次，对应 ChatModelAgentConfig.MaxIterations。
	MaxIterations int `yaml:"max_iterations,omitempty" json:"max_iterations,omitempty"`

	// WorkingDirectory 工作目录（绝对路径），供 eino-ext Local Backend、工具路径约束等使用。
	WorkingDirectory string `yaml:"working_directory,omitempty" json:"working_directory,omitempty"`

	// Tools 工具与中间件相关（filesystem 基于 adk/middlewares/filesystem + filesystem.Backend）。
	Tools ToolsSpec `yaml:"tools,omitempty" json:"tools,omitempty"`

	// Middlewares 中间件列表标识（具体构造在代码中按名称注册 ChatModelAgentMiddleware）。
	Middlewares []MiddlewareRef `yaml:"middlewares,omitempty" json:"middlewares,omitempty"`

	// ModeConfig 按 Mode 附加的结构化配置（多 agent、deep、plan-execute 等）。
	ModeConfig *ModeConfig `yaml:"mode_config,omitempty" json:"mode_config,omitempty"`
}

// AgentMode 与 eino ADK 提供的组合 primitive 对齐。
type AgentMode string

const (
	ModeReAct         AgentMode = "react"
	ModeDeep          AgentMode = "deep"
	ModeSequential    AgentMode = "sequential"
	ModeParallel      AgentMode = "parallel"
	ModeLoop          AgentMode = "loop"
	ModeSupervisor    AgentMode = "supervisor"
	ModePlanExecute   AgentMode = "plan_execute"
	ModeCustom        AgentMode = "custom"
)

// ModeConfig 各模式特有字段；未使用的字段应为空。
type ModeConfig struct {
	// Deep 对应 deep.Config 中可由配置驱动的常用项（其余在代码中补全）。
	Deep *DeepModeConfig `yaml:"deep,omitempty" json:"deep,omitempty"`

	// SubAgentNames 顺序/并行/循环/supervisor 等模式下子 agent 的逻辑名称列表（实例在代码中绑定）。
	SubAgentNames []string `yaml:"sub_agent_names,omitempty" json:"sub_agent_names,omitempty"`

	// PlanExecute 计划-执行-再规划角色名称引用（具体 Agent 实例由注册表解析）。
	PlanExecute *PlanExecuteModeConfig `yaml:"plan_execute,omitempty" json:"plan_execute,omitempty"`

	// Supervisor supervisor.Config 中与配置相关的片段。
	Supervisor *SupervisorModeConfig `yaml:"supervisor,omitempty" json:"supervisor,omitempty"`

	// LoopMaxIterations Loop Agent 最大轮数。
	LoopMaxIterations int `yaml:"loop_max_iterations,omitempty" json:"loop_max_iterations,omitempty"`
}

type DeepModeConfig struct {
	WithoutWriteTodos        bool `yaml:"without_write_todos,omitempty" json:"without_write_todos,omitempty"`
	WithoutGeneralSubAgent   bool `yaml:"without_general_sub_agent,omitempty" json:"without_general_sub_agent,omitempty"`
	MaxIteration             int  `yaml:"max_iteration,omitempty" json:"max_iteration,omitempty"`
}

type PlanExecuteModeConfig struct {
	PlannerName   string `yaml:"planner,omitempty" json:"planner,omitempty"`
	ExecutorName  string `yaml:"executor,omitempty" json:"executor,omitempty"`
	ReplannerName string `yaml:"replanner,omitempty" json:"replanner,omitempty"`
}

type SupervisorModeConfig struct {
	SupervisorAgentName string `yaml:"supervisor_agent,omitempty" json:"supervisor_agent,omitempty"`
}

// ToolsSpec 声明本 agent 可用的工具集合。
type ToolsSpec struct {
	// Filesystem 文件系统工具族（read_file / write_file / …），由 filesystem 中间件注入。
	Filesystem *FilesystemToolsSpec `yaml:"filesystem,omitempty" json:"filesystem,omitempty"`

	// ReturnDirectly 工具名 -> true 时与 ChatModelAgent ToolsConfig.ReturnDirectly 对齐。
	ReturnDirectly map[string]bool `yaml:"return_directly,omitempty" json:"return_directly,omitempty"`
}

// FilesystemToolsSpec 对应 MiddlewareConfig 中 Backend / Shell 与 Local 配置。
type FilesystemToolsSpec struct {
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Backend local | in_memory | agentkit_sandbox（实现来自 eino / eino-ext）
	Backend string `yaml:"backend" json:"backend"`

	// Shell 是否注册 execute（同步）；与 StreamingShell 互斥。
	Shell bool `yaml:"shell,omitempty" json:"shell,omitempty"`

	// StreamingShell 是否使用流式 shell（与 Shell 互斥）。
	StreamingShell bool `yaml:"streaming_shell,omitempty" json:"streaming_shell,omitempty"`

	// Local 专用于 eino-ext local backend（路径校验、命令校验等）。
	Local *LocalBackendConfig `yaml:"local,omitempty" json:"local,omitempty"`
}

// LocalBackendConfig 映射 eino-ext/adk/backend/local.Config 的可配置子集。
type LocalBackendConfig struct {
	// ValidateCommand 在代码侧由用户提供函数；此处仅保留开关，true 表示启用内置白名单文件或自定义钩子名。
	StrictCommands bool `yaml:"strict_commands,omitempty" json:"strict_commands,omitempty"`
}

// MiddlewareRef 由名称引用已注册的 ChatModelAgentMiddleware 构造器。
type MiddlewareRef struct {
	Name string         `yaml:"name" json:"name"`
	With map[string]any `yaml:"with,omitempty" json:"with,omitempty"`
}

// ModelsSpec 多模型注册与当前选用。
type ModelsSpec struct {
	// Providers 按引用名注册多个厂商/协议。
	Providers map[string]ProviderConfig `yaml:"providers" json:"providers"`

	// Active 当前使用的 provider 名与模型名。
	Active ModelRef `yaml:"active" json:"active"`
}

// ModelRef 指向某一 Provider 及其模型标识。
type ModelRef struct {
	Provider string `yaml:"provider" json:"provider"`
	Model    string `yaml:"model" json:"model"`
}

// ProviderConfig 描述与 eino-ext 各 model 子包对齐的驱动参数（仅占位，具体 Option 在代码中映射）。
type ProviderConfig struct {
	// Driver 逻辑驱动名：openai | claude | gemini | ollama | ark | qwen | deepseek | openrouter | …
	Driver string `yaml:"driver" json:"driver"`

	// APIKeyEnv 环境变量名，避免在配置文件中写明文密钥。
	APIKeyEnv string `yaml:"api_key_env,omitempty" json:"api_key_env,omitempty"`

	BaseURL string `yaml:"base_url,omitempty" json:"base_url,omitempty"`

	// Extra 透传厂商特有字段（temperature、top_p、by_azure 等）。
	Extra map[string]any `yaml:"extra,omitempty" json:"extra,omitempty"`
}

// MemorySpec 业务侧「记忆」策略，与 eino 的 History / Session / 外部向量库分工。
type MemorySpec struct {
	// SessionStore 会话 KV 的持久化后端：memory | file | redis（实现由业务层完成）。
	SessionStore string `yaml:"session_store,omitempty" json:"session_store,omitempty"`

	SessionFilePath string `yaml:"session_file_path,omitempty" json:"session_file_path,omitempty"`

	// MaxContextMessages 注入模型前的最大消息条数（业务层裁剪 History / messages）。
	MaxContextMessages int `yaml:"max_context_messages,omitempty" json:"max_context_messages,omitempty"`

	// OutputKey 对应 ChatModelAgentConfig.OutputKey，写入 SessionValues。
	OutputKey string `yaml:"output_key,omitempty" json:"output_key,omitempty"`
}

// ObservabilitySpec 对应 callbacks 与 eino-ext 集成。
type ObservabilitySpec struct {
	// GlobalHandlers 要装配的 handler 插件名列表，如 langfuse | cozeloop | apmplus。
	GlobalHandlers []string `yaml:"global_handlers,omitempty" json:"global_handlers,omitempty"`

	// LogLevel 业务日志级别。
	LogLevel string `yaml:"log_level,omitempty" json:"log_level,omitempty"`
}

// InterruptSpec 对应 Runner + Checkpoint。
type InterruptSpec struct {
	Enabled bool `yaml:"enabled" json:"enabled"`

	// CheckpointBackend memory | file（file 时使用 CheckpointFilePath 持久化字节）。
	CheckpointBackend string `yaml:"checkpoint_backend,omitempty" json:"checkpoint_backend,omitempty"`

	// CheckpointFilePath CheckPointStore 为 file 时使用的目录或文件前缀。
	CheckpointFilePath string `yaml:"checkpoint_file_path,omitempty" json:"checkpoint_file_path,omitempty"`
}
