# Eino Agent 执行稳定性：错误、中断与恢复 — 处理思路与最佳实践

本文基于 Eino 官方文档（ADK Runner、ChatModelAgent、v0.7 中断恢复重构等）整理：**运行时错误**、**主动中断**、**从检查点恢复** 三条主线的机制与工程实践。实现细节以你所依赖的 **`eino` 版本** 为准。

---

## 一、问题分层：先区分「失败」与「暂停」

| 类型 | 典型情况 | 框架侧常见能力 |
|------|----------|------------------|
| **可重试瞬时错误** | 429、网络抖动、流式中途异常后重试 | `ChatModelAgent` 的 **`ModelRetryConfig`**、流式错误中的 **`WillRetryError`** |
| **业务/工具错误** | 工具返回 err、参数不合法 | 工具实现返回明确错误；中间件 **`WrapInvokableToolCall`** 等可把错误转成模型可读的反馈；**不要**把「中断信号」与普适错误混包（见 v0.7.1+ 说明） |
| **逻辑上限** | ReAct 死循环、模型反复调用工具 | **`MaxIterations`**、产品层策略（限流、拒绝） |
| **主动中断（HITL）** | 需人工确认、补全外部输入 | **`AgentEvent.Action.Interrupted`** + **`CheckPointStore`** + **`Resume`** |
| **可观测** | 排障、审计 | **`callbacks.Handler`** 的 `OnError`、全链路日志 |

---

## 二、模型调用失败：重试与流式

### 1. `ModelRetryConfig`（ChatModelAgent）

- 在 **`ChatModelAgentConfig`** 中配置后，可对 **ChatModel 调用失败**（含流式响应过程中的部分失败策略）按策略自动重试。
- **流式场景**：文档说明在消费 **`AgentEvent`** 里的流时，可能收到 **`WillRetryError`**，表示 **后续还会重试**，用于 UI 区分「暂时失败」与「最终失败」。

**实践：** 对 429/5xx 设置退避与上限；对 **不可重试** 的业务错误（如 400 内容政策）在策略层直接判为终态，避免无意义重试。

### 2. `MaxIterations`

- 限制 ReAct **推理-工具** 轮数，防止异常循环耗尽配额；默认有上限（如文档中的 20），生产应显式配置并监控触发次数。

---

## 三、中断与恢复（Human-in-the-loop / 长等待）

### 1. 机制概要（ADK Runner）

仅通过 **`adk.NewRunner`** 运行 Agent 时，才可使用 **中断 / 恢复** 等扩展能力（见 [Agent Runner and Extension](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_extension/)）。

三要素：

1. **Interrupted Action**：运行中产出带 **`Action.Interrupted`（`InterruptInfo`）** 的 **`AgentEvent`**，Runner 识别为中断。
2. **Checkpoint**：若配置了 **`RunnerConfig.CheckPointStore`**，且在运行选项中传入 **`WithCheckPointID`**，Runner 会将 **当前运行状态**（含输入、历史等）与 **`InterruptInfo`** 以 **CheckPointID** 为键持久化。
3. **Resume**：调用 **`Runner.Resume(ctx, checkPointID, opts...)`**；被中断的 Agent 需实现 **`ResumableAgent`**（在 **`Agent`** 上增加 **`Resume(ctx, *ResumeInfo, ...)`**）。**`ResumeInfo`** 携带此前的 **`InterruptInfo`** 与 **`EnableStreaming`** 等配置。

**实践：**

- 为每次需要可恢复的对话/任务使用 **稳定且唯一的 `CheckPointID`**（如业务会话 id + 版本）。
- **`InterruptInfo.Data`** 存放对调用方可读的原因、表单 schema、待审批参数等；**Resume** 时通过 **`AgentRunOption`** 把用户输入或审批结果传回（具体 Option 以文档与示例为准）。

### 2. 序列化与自定义类型

- Checkpoint 使用 **gob** 序列化；**自定义类型**需 **`gob.Register` / `gob.RegisterName`**（文档推荐具名注册），避免类型路径或改名导致无法恢复。

### 3. v0.7+ 编排层增强（compose / Graph / Tool）

[v0.7 中断恢复重构说明](https://www.cloudwego.io/docs/eino/release_notes_and_migration/eino_v0.7._-interrupt_resume_refactor/) 提到：

- **`GetInterruptState[T]`** / **`GetResumeContext`**：类型安全地取上次中断状态、判断当前组件是否为恢复目标。
- **两种恢复策略**：隐式「一键恢复全部中断点」 vs **显式「按目标点恢复」**（文档推荐后者）。
- **Graph / Tool 节点**、**嵌套 Agent**、**CompositeInterrupt** 等在后续小版本持续加固；升级后应重点回归 **中断—恢复—再执行** 路径。

---

## 四、工具与中间件层的稳定性

### 1. 工具错误 vs 中断

- 发行说明提到：**工具错误处理** 与 **中断错误** 应区分：例如 **不要再包装 interrupt error**，以免破坏 Runner 对中断的识别。工具作者应 **返回清晰错误信息**，必要时由上层转换为模型可理解的 **Tool 消息**，便于模型自纠。

### 2. 中间件

- **`ChatModelAgentMiddleware`**（如 `WrapInvokableToolCall`）可在 **不破坏主流程** 的前提下做限流、审计、把错误格式化为「给模型的 observation」。

### 3. 幂等与副作用

- 对 **写操作、支付、删文件** 等：在业务层做 **幂等键**、**确认步骤** 或 **先中断再 Resume 执行**，避免模型重复调用造成事故。

---

## 五、可观测与运维

- **`callbacks.OnError`**：记录组件失败时的错误与 **RunInfo**（节点名、组件类型），便于与 trace id 关联。
- **全局 Handler** 仍在 `main`/`TestMain` 初始化一次，避免并发问题。
- 流式路径：**关闭**回调中的 `StreamReader`，避免泄漏与假死。

---

## 六、最佳实践清单（简版）

1. **Runner**：需要 **中断/恢复** 或统一治理时，用 **Runner + CheckPointStore**，不要用裸 `Agent.Run` 却假设能恢复。  
2. **CheckPointID + gob**：自定义状态类型必须 **注册**；Checkpoint 存储选 **可靠后端**（内存仅适合开发）。  
3. **模型错误**：**ModelRetryConfig** + 流式中处理 **`WillRetryError`**；**MaxIterations** 防死循环。  
4. **工具**：错误信息可读；敏感操作配合 **中断/审批**；注意 **interrupt 与普适 err** 语义分离。  
5. **升级**：跨 v0.7 升级时阅读 **interrupt/resume 迁移说明**，并跑通 **Graph / 嵌套 Agent / Tool** 的中断用例。  
6. **产品层**：对用户展示「重试中 / 需人工输入 / 已失败终态」，与框架事件对齐。

---

## 七、参考链接

- [Eino ADK: Agent Runner and Extension（中断、Checkpoint、Resume）](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_extension/)  
- [Eino ADK: ChatModelAgent（ModelRetryConfig、WillRetryError 示例）](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)  
- [v0.7.* interrupt/resume refactor](https://www.cloudwego.io/docs/eino/release_notes_and_migration/eino_v0.7._-interrupt_resume_refactor/)  
- 示例：[eino-examples interrupt/resume 文档](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/docs/ch07_interrupt_resume.md)  
- 相关：[eino-observability.md](./eino-observability.md)、[eino-memory.md](./eino-memory.md)、[eino-agents-and-practices.md](./eino-agents-and-practices.md)
