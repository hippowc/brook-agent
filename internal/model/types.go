package model

// AgentRequest 是整个 Agent 执行链路的标准输入。
// entry 层将不同协议的请求统一转成该结构，再交给 frame/core 处理。
type AgentRequest struct {
	SessionID string            `json:"session_id"`
	Input     string            `json:"input"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// AgentResponse 是整个 Agent 执行链路的标准输出。
type AgentResponse struct {
	SessionID string `json:"session_id"`
	Output    string `json:"output"`
	Finished  bool   `json:"finished"`
	TraceID   string `json:"trace_id,omitempty"`
}

// Message 参考 OpenAI Chat 风格定义，作为 memory 的核心数据结构。
type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	Name       string     `json:"name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

// ToolCall 表示模型发起的一次工具调用请求。
type ToolCall struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Args     map[string]string `json:"args,omitempty"`
	RawInput string            `json:"raw_input,omitempty"`
}

// ToolResult 表示工具执行返回结果。
type ToolResult struct {
	CallID  string `json:"call_id"`
	Name    string `json:"name"`
	Output  string `json:"output"`
	IsError bool   `json:"is_error"`
}

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)
