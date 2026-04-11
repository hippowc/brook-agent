# Eino / Eino-Ext 接口总结

本文基于 [cloudwego/eino](https://github.com/cloudwego/eino) 与 [cloudwego/eino-ext](https://github.com/cloudwego/eino-ext) 源码结构整理：**`eino` 模块提供框架契约（核心接口）**；**`eino-ext` 提供面向具体厂商/基础设施的实现与集成扩展（业务与生态接口）**。  
说明：下列接口名以 Go 包路径区分，避免同名混淆；版本演进以各仓库当前主分支为准。

---

## 一、`github.com/cloudwego/eino`：核心接口（契约层）

### 1. 编排与执行（`compose`）

| 接口 | 职责摘要 |
|------|-----------|
| **`Runnable[I, O any]`** | 可执行单元的统一抽象。提供 `Invoke` / `Stream` / `Collect` / `Transform` 四种数据流形态，Graph/Chain 编译后均落在此模型上。 |
| **`AnyGraph`** | 内部用于标识可编译的图（`Graph`、`Chain` 等）的统一能力（类型、编译入口）。业务侧主要使用具体 Graph API，较少直接实现该接口。 |
| **`Serializer`** | Checkpoint 持久化：`Marshal(v any) ([]byte, error)`、`Unmarshal(data []byte, v any) error`。 |
| **`GraphCompileCallback`** | 图编译结束回调：`OnFinish(ctx, *GraphInfo)`，用于观测/提取节点元数据。 |

**类型别名（对外导出）：** `CheckPointStore`（与 `internal/core` 中的 checkpoint 存储一致），通过 `compose.WithCheckPointStore` 等选项注入。

### 2. 组件分类与可观测（`components`）

| 接口 | 职责摘要 |
|------|-----------|
| **`Typer`** | `GetType() string`：为组件提供可读实现名（调试、DevOps、工具推断名等）。 |
| **`Checker`** | `IsCallbacksEnabled() bool`：声明是否由组件自行驱动 callback，关闭框架默认包裹。 |

常量 **`components.Component`** 标识组件类别（如 `ChatModel`、`Embedding`、`Tool`、`Retriever` 等），供回调与工具链识别。

### 3. 模型与工具（`components/model`、`components/tool`）

| 接口 | 职责摘要 |
|------|-----------|
| **`BaseChatModel`** | `Generate`（整段输出）与 `Stream`（流式 `*schema.StreamReader[*schema.Message]`）。 |
| **`ChatModel`**（Deprecated） | 在 `BaseChatModel` 上增加 `BindTools`（原地绑定，并发不安全）。 |
| **`ToolCallingChatModel`** | 推荐路径：`WithTools` 返回新实例，线程更安全。 |
| **`BaseTool`** | `Info` 产出 `*schema.ToolInfo`（名称、描述、参数 JSON Schema）。 |
| **`InvokableTool` / `StreamableTool`** | 同步/流式执行工具调用（参数为 JSON 字符串或流式字符串块）。 |
| **`EnhancedInvokableTool` / `EnhancedStreamableTool`** | 结构化多模态参数与 `schema.ToolResult` 结果（及流式变体）。 |

### 4. RAG 与文档管线（`components/embedding|retriever|indexer|document|prompt`）

| 接口 | 职责摘要 |
|------|-----------|
| **`embedding.Embedder`** | `EmbedStrings`：文本批量转向量。 |
| **`retriever.Retriever`** | `Retrieve`：按查询返回 `[]*schema.Document`。 |
| **`indexer.Indexer`** | `Store`：写入文档并返回后端 ID。 |
| **`document.Loader`** | `Load`：从 `Source.URI` 拉取并产出文档。 |
| **`document.Transformer`** | `Transform`：切分、过滤、重排等文档变换。 |
| **`document/parser.Parser`** | `Parse`：从 `io.Reader` 解析为 `[]*schema.Document`。 |
| **`prompt.ChatTemplate`** | `Format(ctx, vs map[string]any, opts...)`：渲染为 `[]*schema.Message`（与下文 `schema.MessagesTemplate` 并存于不同层次）。 |

### 5. 数据模型与解析（`schema`）

| 接口 | 职责摘要 |
|------|-----------|
| **`MessagesTemplate`** | `Format(ctx, vs, formatType)`：消息模板渲染（如 `MessagesPlaceholder` 占位）。 |
| **`MessageParser[T any]`** | `Parse(ctx, *Message) (T, error)`：把模型消息解析为强类型结果。 |

流式相关另有内部/辅助 `reader` 接口（一般业务不直接实现）。

### 6. 回调（`callbacks`，对 `internal/callbacks` 的导出别名）

| 接口 | 职责摘要 |
|------|-----------|
| **`Handler`** | `OnStart` / `OnEnd` / `OnError` / `OnStartWithStreamInput` / `OnEndWithStreamOutput`。 |
| **`TimingChecker`** | `Needed`：声明本次调用需要哪些回调时机，用于削减流拷贝与协程开销。 |

### 7. Agent（`adk`）

| 接口 | 职责摘要 |
|------|-----------|
| **`Agent`** | `Name` / `Description` / `Run`（返回异步事件迭代器 `AsyncIterator[*AgentEvent]`）。 |
| **`OnSubAgents`** | 子 Agent 挂载/转移相关生命周期钩子。 |
| **`ResumableAgent`** | 在 `Agent` 之上增加 `Resume`，支持中断恢复。 |
| **`ChatModelAgentMiddleware`** | 围绕 ChatModel Agent 的中间件扩展点（详见 `adk/handler.go` 等）。 |

ADK 另含 **filesystem** 场景的 `Backend` / `Shell` / `StreamingShell`，以及中间件内部的 `Backend`、`AgentHub`、`ModelHub` 等（面向扩展作者）。

### 8. 预置与流程（`flow`、`adk/prebuilt` 等）

- **`flow/agent/multiagent/host.MultiAgentCallback`**：多 Agent 主机侧回调。
- **`flow/agent/react/option.MessageFuture`**：ReAct 相关异步消息约定。
- **`adk/prebuilt/planexecute.Plan`**：Plan-Execute 等预置模式中的计划抽象。

### 9. 内部包中的接口（了解即可）

`internal/core` 的 **`CheckPointStore`**、**`InterruptContextsProvider`**，`internal/callbacks` 的 **`Handler`** / **`TimingChecker`** 等：由 `compose`/`callbacks` 对外再导出或包装，应用代码优先使用公开路径。

---

## 二、`eino-ext`：业务与生态接口（实现与集成层）

`eino-ext` **按子目录拆分多个独立 Go module**（各组件目录下自有 `go.mod`），在 `go.mod` 中依赖 `github.com/cloudwego/eino`，用**具体 struct 实现**上一节的 `BaseChatModel`、`Retriever`、`Indexer` 等核心接口。  
除实现外，扩展库会额外声明少量**集成用接口**，便于插件化或对接外部系统。

### 1. 组件实现（映射到 `eino` 核心接口）

下表概括「实现哪类核心接口」与典型目录（子模块名随版本以仓库为准）：

| 核心契约（eino） | eino-ext 中的典型实现方向 |
|------------------|---------------------------|
| `BaseChatModel` / `ToolCallingChatModel` | `components/model/*`：OpenAI、Claude、Gemini、Ollama、Qwen、DeepSeek、Ark、OpenRouter、Qianfan 等 |
| `Embedder` | `components/embedding/*`：OpenAI、Ollama、Gemini、DashScope、Ark、Qianfan、TencentCloud 等 |
| `Retriever` / `Indexer` | 向量库与搜索：`milvus`/`milvus2`、`qdrant`、`redis`；Elasticsearch `es7`/`es8`/`es9`；OpenSearch `opensearch2`/`opensearch3`；火山 `volc_vikingdb`、`volc_knowledge`；`dify` 等 |
| `Loader` / `Parser` / `Transformer` | `components/document/loader|parser|transformer`：本地文件、URL、S3、PDF/DOCX/HTML/XLSX、切分与 rerank 等 |
| `ChatTemplate` 或模板生态 | `components/prompt/*`：如 MCP、Cozeloop 等集成 |
| `tool.*` 系列 | `components/tool/*`：Google/Bing/DuckDuckGo 搜索、HTTP、MCP、浏览器、命令行、Wikipedia 等 |

**Embedding 缓存层（在 `Embedder` 之上做包装）** 额外定义：

- `components/embedding/cache`：`Cacher`、`Generator`、`Option` 等，用于缓存向量结果。

**部分检索器** 为不同查询模式定义 **`SearchMode` 接口**（如 ES、OpenSearch、Milvus 子包中的 `search_mode`），属于**后端检索策略**扩展，而非 `eino` 核心契约。

### 2. 回调与可观测（`callbacks`）

实现或封装 `eino/callbacks.Handler` 的集成，例如：

- `callbacks/langfuse`、`callbacks/langsmith`、`callbacks/cozeloop`、`callbacks/apmplus` 等。

部分包定义解析型小接口，例如 **`CallbackDataParser`**（Cozeloop 数据解析）或对接外部的 **`Langsmith`** / **`Langfuse`** 抽象，用于把 Eino 回调数据映射到对应 SaaS。

### 3. 库与工具（`libs`、`devops`）

- **`libs/acl/*`**：如 OpenTelemetry 选项接口 `Option`、Langfuse 客户端抽象 `Langfuse` 等，为回调与追踪提供适配层。
- **`devops`**：可视化调试、容器与调试运行等服务接口（如内部 `ContainerService`、`DebugService`），面向 IDE 插件与本地调试，而非 LLM 推理契约。

### 4. 其他目录

- **`adk/backend`**：与 ADK 后端能力相关的扩展实现。
- **`skills/*`**：面向 Agent/组件/编排的 skill 模板或引导项目（非核心 `eino` 接口定义）。

---

## 三、如何阅读源码时的分工记忆

1. **先找 `eino` 里带 `interface` 的包**：`compose`、`components/*`、`schema`、`callbacks`、`adk` —— 这是**能替换实现、能插拔**的边界。  
2. **再在 `eino-ext` 搜具体厂商目录**：同一类能力（如 ChatModel）往往有多份 `go.mod`，按需只引用一个子模块。  
3. **`eino-ext` 里除「实现 struct」外，名字像 `SearchMode`、`Cacher`、`Langfuse` 的多为「对接外部系统」的辅助接口**，与业务选型的耦合更高。

---

## 四、参考链接

- Eino 文档：<https://www.cloudwego.io/docs/eino/overview/>  
- Eino 组件说明：<https://www.cloudwego.io/docs/eino/core_modules/components/>  
- Eino 生态集成：<https://www.cloudwego.io/docs/eino/ecosystem_integration/>
