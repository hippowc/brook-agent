# 内部模块划分（与 `req/agent-req.md` 对齐）

| 目录 | 职责 |
|------|------|
| `internal/core/model` | 按配置构造 `ToolCallingChatModel`（OpenAI / Ollama 等，扩展时在此增加 driver） |
| `internal/core/fs` | 构造 `filesystem` Backend 与 **FileSystem 中间件**（local / in_memory） |
| `internal/core/agent` | 按 `agent.mode` 组装 **ChatModelAgent / Deep / Sequential / Parallel / Loop / Supervisor / PlanExecute** |
| `internal/business/store` | 业务层持久化：**文件型 Session**、**文件型 CheckPointStore**（非 eino 内置，对接 Runner） |
| `internal/extension/callbacks` | 全局 **callbacks.Handler** 示例（可替换为 eino-ext APMPlus / CozeLoop 等） |
| `internal/extension/middleware` | **ChatModelAgentMiddleware** 注册表（按名称扩展） |
| `pkg/agentconfig` | 配置模型与校验 |
| `pkg/a2ui` | **A2UI 风格 JSONL** 事件导出（流式 UI 协议子集） |
| `cmd/brook` | 可执行入口：加载 YAML、**Runner**、可选 **Resume**、A2UI 输出 |
