# Eino：可观测性（Observability）与实践

本文说明 **编排层（compose + callbacks）** 与 **ADK** 中如何通过 **Callbacks** 做日志、链路追踪、指标与调试，而不侵入业务逻辑。细节以官方文档与当前版本为准。

**相关：** 记忆与状态（History、Session、Checkpoint）见 [eino-memory.md](./eino-memory.md)。

---

## 1. 核心机制：Callbacks（横切能力）

Eino 用 **Callback** 把日志、链路追踪、指标、调试展示等与业务组件解耦：**组件 / Graph 节点 / Graph 自身** 在固定的 **Callback Timing** 调用用户注册的 **Handler**，并传入 **RunInfo** 与 **CallbackInput/Output**（或流式版本）。

| 时机 | 含义 |
|------|------|
| `OnStart` | 非流式输入，开始执行前 |
| `OnEnd` | 成功结束，返回前 |
| `OnError` | 返回错误前 |
| `OnStartWithStreamInput` | 输入为流 |
| `OnEndWithStreamOutput` | 输出为流 |

**RunInfo** 包含：节点/业务名（`Name`）、实现类型（`Type`，如实现 `Typer`）、组件类别（`Component`，如 `ChatModel`、`Tool`）。

**注入方式：**

- **`callbacks.AppendGlobalHandlers`**：进程级全局 Handler，适合统一追踪/日志；**非并发安全**，官方建议在 **服务初始化阶段只注册一次**。
- **`compose.WithCallbacks`**：单次 Graph 运行期注入；可指定整图或某个节点（含嵌套图路径）。
- **`callbacks.InitCallbacks` / `ReuseHandlers`**：不用 Graph、单独跑组件时使用。

**流式路径：** Handler 若消费 `*schema.StreamReader`，必须在用完后 **关闭**，否则易造成协程/内存泄漏；可通过实现 **`TimingChecker.Needed`** 跳过不需要的时机，减少流拷贝开销。

详见：[Callback User Manual](https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/callback_manual/)、Quick Start [Chapter 6](https://www.cloudwego.io/docs/eino/quick_start/chapter_06_callback_and_trace/)。

---

## 2. 类型化观测：按组件解析 Input/Output

`CallbackInput` / `CallbackOutput` 底层为 `any`。实际处理时应根据 **`RunInfo.Component`**（或 `Type`）过滤，并用各组件包提供的转换函数（如 ChatModel 的 **`ConvCallbackInput` / `ConvCallbackOutput`**）安全断言，避免把无关节点的回调当模型回调处理。

---

## 3. ADK 与 Callback 的关系

**ChatModelAgent** 在 ADK 概述中的说明是：在 ReAct 执行过程中通过 **`callbacks.Handler`** 导出过程，再转换为 **`AgentEvent`** 给调用方消费。因此 **Agent 侧可观测** 既可以通过 **全局/图 Callback** 做细粒度组件追踪，也可以通过消费 **`AsyncIterator[*AgentEvent]`** 做产品层展示（流式消息、工具结果、错误等）。

---

## 4. 生态与工具链（eino-ext / 插件）

| 方向 | 说明 |
|------|------|
| **Eino Dev 插件** | IDE 侧可视化编排、调试等，降低「黑盒」感。见 [Eino Dev: Application Tooling](https://www.cloudwego.io/docs/eino/core_modules/devops/)。 |
| **官方 Callback 集成组件** | 文档列表包含如 **APMPlus**、**CozeLoop** 等（见 [Callbacks 生态](https://www.cloudwego.io/docs/eino/ecosystem_integration/callbacks/)，以各子模块 README 为准）。 |
| **其他追踪后端** | 社区/仓库中常见 **Langfuse、LangSmith** 等适配，通过实现同一套 **`callbacks.Handler`** 接入。 |

---

## 5. 实践要点

1. **初始化阶段** 注册全局 Handler，避免运行中并发修改 Handler 列表。  
2. **按 RunInfo 过滤**，只对关心的 `Component`/节点名打点。  
3. **流式回调** 必须 **读完并关闭** `StreamReader`。  
4. **Token/延迟**：优先从 **ChatModel 的 CallbackOutput**（如含 `TokenUsage`）取数，与业务日志字段对齐。  
5. **GraphCompileCallback**：需要编译期图结构观测时，使用 `compose` 的 **`OnFinish`** 回调拿到 `GraphInfo`（节点类型、实例等）。  
6. 用 **全局 Handler** 做 tracing/logging，用 **图级/节点级 WithCallbacks** 做细粒度实验或对比；对流与指标敏感的路径实现 **`TimingChecker`**。  
7. 结合 **Eino Dev** 做开发与联调，生产用 **Callback 导出到 APM/Tracing 平台**。

---

## 6. 文档交叉引用

- [eino-interfaces.md](./eino-interfaces.md)（`Handler`、`Runnable`、组件回调）  
- [eino-agents-and-practices.md](./eino-agents-and-practices.md)（Agent 模式与 Runner）  
- [eino-memory.md](./eino-memory.md)（History、Session、Checkpoint）

---

## 7. 参考链接

- [Callback User Manual](https://www.cloudwego.io/docs/eino/core_modules/chain_and_graph_orchestration/callback_manual/)  
- [Callbacks 生态集成列表](https://www.cloudwego.io/docs/eino/ecosystem_integration/callbacks/)  
- [Eino Dev](https://www.cloudwego.io/docs/eino/core_modules/devops/)  
