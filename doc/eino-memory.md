# Eino：记忆与状态（History / Session / Checkpoint）

本文说明 Eino **ADK** 与 **compose** 中与「记忆」相关的概念：**不等于**单独内置一套通用向量记忆库；长期语义记忆通常需 **Retriever + 外部存储** 或业务层方案。细节以官方文档与当前版本为准。

**相关：** 可观测性（Callbacks、追踪）见 [eino-observability.md](./eino-observability.md)。

---

## 1. 框架里「记忆」指什么

与「记忆」相关的概念主要是 **对话与协作状态**、**单次运行内 KV**、以及 **可持久化的执行断点**：

| 概念 | 作用域 | 典型用途 |
|------|--------|----------|
| **History** | 多 Agent 一次协作链 | 把上游 Agent 产生的事件转成后续 Agent 的输入 |
| **SessionValues** | 单次 Run 内 | 跨 Agent 共享短生命周期变量（用户 id、计划、结构化中间结果） |
| **Checkpoint / Resume** | 同一次运行生命周期内可恢复 | 人机协同、审批、长等待后再继续 |
| **Retriever / Graph** | 按请求检索 | 外部知识库、「可检索记忆」 |

---

## 2. History（多 Agent 协作下的「发生了什么」）

- 多 Agent 运行中，各 Agent 产出的 **`AgentEvent`** 会进入 **History**。  
- 后续 Agent 构建 **`AgentInput`** 时，会把此前 History **转换并拼接**进输入（默认将其他 Agent 的 Assistant/Tool 消息转为当前模型易消费的形态），等价于让模型看到「上游发生了什么」。  
- **`RunPath`** 标识事件来自哪条 Agent 执行路径，用于区分来源，不额外承担存储职责。  
- 若需裁剪或改写再喂给模型，可使用 **`WithHistoryRewriter`** 自定义从 History 到消息列表的规则。

详见：[Eino ADK: Agent Collaboration — History](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_collaboration/)。

---

## 3. SessionValues（单次运行内的 KV「会话态」）

- **SessionValues** 是挂在 **Context** 上的 **临时 KV**，用于 **同一次 Run** 内跨 Agent 共享状态（例如用户 id、本轮计划、中间结构化结果）。  
- API：**`GetSessionValues` / `GetSessionValue` / `AddSessionValue` / `AddSessionValues`**。  
- **注意：** Runner 运行时会 **重新初始化 Context**，因此在 **`Run` 之外** 预先 `AddSessionValue` **不会生效**；若要在 Agent 启动前注入，应使用 **`adk.WithSessionValues`**（作为 **`AgentRunOption`**）。  
- **ChatModelAgent** 的 **`OutputKey`**：若配置，会将本轮最终回复写入 Session（文档描述为通过 `AddSessionValue` 与 `outputKey` 关联），便于后续节点或 Instruction 模板引用。  
- **Instruction 模板**：默认 **`GenModelInput`** 可把 **SessionValues** 渲染进系统提示（如 `{Time}`、`{User}` 占位），与「短期会话变量」天然结合。

---

## 4. Checkpoint / Resume（持久化运行状态，非纯「聊天记忆」）

- 通过 **`adk.NewRunner`** 配置 **`CheckPointStore`（compose.CheckPointStore）**，支持 **中断—恢复**（如人机协同、长等待）。  
- 与 **Session** 的关系：概述文档强调 **中断与恢复仍属于同一次运行生命周期** 内的能力；具体序列化字段以 **`AgentExtension`** 与实现为准。  
- 这是 **执行状态** 的持久化，不等同于「长期语义记忆」；长期记忆仍宜用 **外部 DB + Retriever** 或业务层缓存。

---

## 5. 与 compose 图的关系（RAG / 外部记忆）

- 在 **Graph** 中，**Retriever、Indexer、ChatTemplate** 等组件负责 **从外部索引读知识**，这是工程上最常见的「可检索记忆」。  
- **Checkpoint（compose）** 更多用于 **图执行断点**（与 ADK Runner 的 CheckPoint 概念相关但使用场景不同，需按包文档区分）。

---

## 6. 实践要点

- **短期、同轮协作**：优先 **SessionValues + OutputKey + Instruction 占位符**；注意 **WithSessionValues** 的注入时机。  
- **多 Agent 叙事**：理解 **History 的默认转换规则**；需要控制长度时用 **HistoryRewriter** 或产品层摘要。  
- **长期知识**：**Retriever + 向量库/搜索**，不要指望 Session 替代 KB。  
- **要可恢复的人机流程**：**Runner + CheckPointStore + Resume**，并设计好中断时写入的 payload（详见 [eino-agent-stability-and-recovery.md](./eino-agent-stability-and-recovery.md)）。

---

## 7. 文档交叉引用

- [eino-interfaces.md](./eino-interfaces.md)  
- [eino-agents-and-practices.md](./eino-agents-and-practices.md)  
- [eino-observability.md](./eino-observability.md)  

---

## 8. 参考链接

- [Eino ADK: Agent Collaboration](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_collaboration/)  
- [Eino ADK: Overview](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_preview/)  
- [Eino ADK: Agent Runner and Extension](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_extension/)（Checkpoint / Resume）  
