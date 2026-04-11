# Eino：可构建的 Agent 类型与最佳实践

本文基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) / [Eino 文档站](https://www.cloudwego.io/docs/eino/overview/) 中 **ADK（Agent Development Kit）**、**预置多 Agent 模式**、以及 **AgenticModel（Beta）** 等公开说明整理，便于选型与落地。更细的接口清单见同目录 [`eino-interfaces.md`](./eino-interfaces.md)。

---

## 一、用该框架可以构建哪些类型的 Agent

下面按「从常用到进阶」分层说明；它们可以组合使用（例如 Workflow Agent 里嵌 `ChatModelAgent`）。

### 1. 单智能体：ChatModelAgent（ReAct）

- **定位**：ADK 中最核心的预置 Agent，封装与 **支持工具调用的对话模型**（`ToolCallingChatModel`）的交互逻辑。
- **内部模式**：经典 **[ReAct](https://react-lm.github.io/)**（Reason → Act → Observe 循环）：调用模型 → 可能产生工具调用 → 执行工具 → 将结果写回上下文 → 再推理，直到模型不再请求工具或满足退出条件。
- **无工具时**：退化为**单次** ChatModel 调用。
- **典型场景**：带搜索/计算/内部 API 的助手、运维排障、分步调研等需要「边想边做」的任务。

参考：[Eino ADK: ChatModelAgent](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)

### 2. 开箱即用：DeepAgents（基于 ChatModelAgent）

- **定位**：在 `ChatModelAgent` 之上提供**默认的规划、子 Agent 委派与上下文管理**，减少自建 Prompt/工具拼装成本（Eino 版本要求见官方文档，如 `>= v0.5.14`）。
- **核心结构**：
  - **Main Agent**：主协调者，仍走 ReAct + 工具调用。
  - **WriteTodos**：内置规划工具，将复杂任务拆成可跟踪的 todo（可按业务调参，避免过度/不足调用）。
  - **TaskTool**：统一入口调用 **SubAgents**，主/子 **上下文隔离**，避免子过程污染主对话。
  - 可选 **文件系统 / Shell** 等能力（通过 `filesystem.Backend`、`Shell` / `StreamingShell` 等配置注入）。
- **典型场景**：多步骤、多角色、需要「项目经理式」分解与委派的复杂任务（官方示例含类似「Excel Agent」类场景）。

参考：[Eino ADK: DeepAgents](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/deepagents/)

### 3. 工作流型：Workflow Agents（顺序 / 并行 / 循环）

- **Sequential Agent**：子 Agent **严格顺序**执行，可传递前置输出；任一子 Agent 触发退出/中断时**整链提前结束**。适合 ETL、流水线式任务。
- **Parallel Agent**：子 Agent **并发**执行、共享初始输入，结束后聚合结果。适合多源采集、多渠道推送等。
- **Loop Agent**：按配置**重复**执行一段 Sequential 子流程，支持最大轮次或 `ExitAction` 结束；结果可跨轮累积。适合同步校验、迭代优化、轮询类任务。

参考：[Eino ADK 设计模式总览](https://www.cloudwego.io/docs/eino/overview/eino_adk0_1/) 中「WorkflowAgents」章节。

### 4. 多智能体预置模式

| 模式 | 要点 | 适用场景 |
|------|------|----------|
| **Supervisor** | 一个 Supervisor 分配任务、汇总子 Agent 结果并决策下一步；子 Agent 完成后**确定性回调**到 Supervisor。 | 需要中心路由的客服、研发协作、研究项目管理等。 |
| **Plan-Execute** | Planner / Executor / Replanner 协作，**计划—执行—再规划**闭环。 | 强步骤感的研究分析、工作流自动化、多工具长链路任务。 |
| **DeepAgents**（上文） | 主 Agent + TaskTool + 子 Agent + WriteTodos，强调**可跳过不必要规划**以控成本。 | 与「纯 Plan-Execute」相比更灵活，但对模型与 Prompt 要求更高。 |

Supervisor 包路径示例：`adk/prebuilt/supervisor`；Plan-Execute：`adk/prebuilt/planexecute`；Deep：`adk/prebuilt/deep`（以当前仓库为准）。

### 5. 自定义 Agent（实现 `adk.Agent`）

- **定位**：任意「可运行智能体单元」都可实现统一接口：`Name` / `Description` / `Run`（返回 `AsyncIterator[*AgentEvent]`）。
- **典型用途**：强定制状态机、与业务系统紧耦合的流程、或非 ChatModel 驱动的执行体。

### 6. 协作机制（可与上述类型组合）

- **Session（KV）**：单次运行内跨 Agent 共享状态（`GetSessionValue` / `AddSessionValue` 等）。
- **Transfer**：将控制权交给命名子 Agent（`NewTransferToAgentAction` 等），适合层级化分工。
- **Agent 即工具**：`NewAgentTool` 把子 Agent 暴露为 **Tool**，由上层 ChatModelAgent 通过工具调用触发；可控制是否向上游透出内部事件（与 `ToolsConfig.EmitInternalEvents` 等配置相关）。

### 7. 模型层：AgenticModel（Beta，自 v0.9 起）

- **定位**：面向 **「目标驱动的自主执行」** 的模型抽象，输入输出以 `AgenticMessage` / `ContentBlock` 为主；部分云厂商在**单次 API 请求内**完成多轮推理与内置工具（如联网搜索），客户端侧与经典「每轮工具都在本地执行」的 ChatModel 流程不同。
- **与 ChatModelAgent 的关系**：前者偏**模型能力与协议演进**；后者是 ADK 里**本地 ReAct 编排**。按供应商能力与产品形态二选一或组合使用。

参考：[AgenticModel User Guide [Beta]](https://www.cloudwego.io/docs/eino/core_modules/components/agentic_chat_model_guide/)

### 8. 非 ADK 但常见：编排型「智能体应用」

- 使用 **`compose` 的 Graph / Chain / Workflow** 将 `ChatTemplate`、`ToolCallingChatModel`、`Retriever`、`Indexer` 等拼成 **RAG、固定 DAG 流程**，语义上也是一种「agentic 应用」，但不等同于实现 `adk.Agent` 的运行单元。需要**可中断恢复、多 Agent 协作、统一事件流**时优先 ADK。

---

## 二、代码与架构层面的最佳实践

### 1. 模型与工具

- **优先使用 `ToolCallingChatModel.WithTools`**，避免使用已废弃且并发不安全的 `ChatModel.BindTools`（见 `components/model` 接口说明）。
- 为 `ChatModelAgent` 配置合理的 **`MaxIterations`**，防止异常循环耗尽配额；按需配置 **`ModelRetryConfig`**，并在消费流式 `AgentEvent` 时识别 **`WillRetryError`**，区分「将重试」与最终失败。
- **`ToolsConfig.ReturnDirectly`**：某些工具执行完后可直接结束 Agent（如「提交工单」类）；多工具同时命中时仅第一个生效——需在工具设计上避免歧义。
- **`Exit` 工具**：需要显式「收束」结束时，可使用 ADK 提供的 `ExitTool` 模式，与 `ReturnDirectly` 类似但语义更清晰。

### 2. Prompt 与输入构造

- **`Instruction` + Session**：默认会把 `Instruction` 与消息列表组合，并支持在 Instruction 中使用 **f-string 风格占位符**（通过 `adk.GetSessionValues()` 等）；若关闭默认行为需自定义 **`GenModelInput`**。
- **默认 `GenModelInput` 使用 pyfmt 类模板**：消息文本中的 `{`、`}` 可能被当作模板语法，需要字面量时按文档要求 **写成 `{{`、`}}`** 转义。
- 对 **DeepAgents** 的 **WriteTodos**：简单任务可能不需要每轮规划；复杂任务则依赖合理分解——可在业务侧补充 Instruction，控制调用频率与粒度。

### 3. 流式与回调

- 处理 **流式 `MessageStream`** 时，遵循文档建议（如 **`SetAutomaticClose`**），避免未消费事件导致 **流未关闭**。
- 全局回调使用 **`AppendGlobalHandlers`** 时应在进程启动阶段 **单次初始化**（文档注明非线程安全）；流式回调路径上 **`StreamReader` 必须关闭**，否则易造成协程/内存泄漏。
- 实现 **`callbacks.TimingChecker`**（`Needed`）可跳过不需要的回调时机，降低流拷贝与开销。

### 4. 可中断 / 可恢复（Human-in-the-loop）

- 使用 **`adk.NewRunner`** 并注入 **`CheckPointStore`**，配合中断事件与 **`Resume`**，在需要人工输入、长等待审批等场景恢复执行。
- 自定义 Agent 若需恢复能力，可实现 **`ResumableAgent`**（在统一 `Agent` 上扩展 `Resume`）。

### 5. 多 Agent 与上下文

- **DeepAgents / TaskTool**：依赖主/子 **上下文隔离**；主 Agent 只应依赖子 Agent **返回结果**，避免假设子 Agent 内部链式思考可见。
- **Supervisor vs Plan-Execute vs DeepAgents**：Supervisor 偏**中心化调度**；Plan-Execute 偏**显式三角色闭环**；DeepAgents 偏 **「规划作工具 + 子 Agent 委派」**，通常延迟与 Token 更高，需按任务复杂度权衡。

### 6. 生态与工程

- **具体厂商实现**（OpenAI、Claude、各类向量库等）放在 **`eino-ext`** 子模块中按需引用，核心契约保持在 **`eino`**，便于替换与测试。
- 需要可视化调试、图编辑时，结合官方 **DevOps** 与 **`GraphCompileCallback`** 等做可观测性（参见 [`eino-interfaces.md`](./eino-interfaces.md)）。

### 7. 文档与版本

- ADK、DeepAgents、AgenticModel 等行为随版本迭代较快，**以当前所选 `eino` 版本对应的官方文档与 Release 说明为准**；升级后应重点回归 **工具调用、中断恢复、流式事件** 路径。

---

## 三、延伸阅读

- [Eino ADK：核心设计模式（总览）](https://www.cloudwego.io/docs/eino/overview/eino_adk0_1/)
- [ToolsNode 与工具使用](https://www.cloudwego.io/docs/eino/core_modules/components/tools_node_guide/)（`ToolsConfig` 复用其配置）
- [示例工程 eino-examples](https://github.com/cloudwego/eino-examples)（含多 Agent、中断恢复等）
