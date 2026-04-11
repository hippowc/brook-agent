# Eino：本机 Local Backend 下的 Tools 清单与 Agent 实践

本文说明在 **本机 Local 文件系统 Backend**（`eino-ext`）配合下，通过 **ADK FileSystem 中间件**（`eino`）向模型暴露的 **默认工具集合**，以及基于这些能力组装 **带 tools 的 Agent** 的推荐做法。  
权威说明见：[FileSystem 中间件](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_filesystem/)、[Local File System / Local Backend](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/filesystem_backend/backend_%E6%9C%AC%E5%9C%B0%E6%96%87%E4%BB%B6%E7%B3%BB%E7%BB%9F/)。

---

## 一、本机 Local 与「工具体」的关系

- **Local Backend** 实现 `github.com/cloudwego/eino/adk/filesystem` 中的 **`Backend`**（以及本机场景下的 **`Shell` / 流式执行** 能力），包路径一般为：  
  `github.com/cloudwego/eino-ext/adk/backend/local`  
- **工具不是 Local 包单独再列一套 API 名称**：模型侧看到的 **`ls` / `read_file` / …** 由 **`adk/middlewares/filesystem`** 根据 `Backend` + 可选 `Shell` **自动注册**；Local 只决定「读写哪块盘、命令怎么执行」。
- **版本提示**：若 `eino` 为 **v0.8.0 及以上**，Local Backend 需使用文档要求的兼容版本（例如文档指向的 [adk/backend/local 发布标签](https://github.com/cloudwego/eino-ext/releases)），避免接口不匹配。

安装示例：

```bash
go get github.com/cloudwego/eino-ext/adk/backend/local
```

---

## 二、通过 FileSystem 中间件注入的 Tools（本机 Local 同样适用）

在 **`MiddlewareConfig.Backend != nil`** 时，中间件会注入下列 **文件类**工具（默认名称如下，均可通过 `*ToolConfig.Name` 改名）：

| 默认工具名 | 作用概要 | 注入条件 |
|------------|----------|----------|
| **`ls`** | 列出目录下文件与子目录 | `Backend` 已配置 |
| **`read_file`** | 读文件内容，支持按行分页（offset / limit） | `Backend` 已配置 |
| **`write_file`** | 创建或覆盖文件 | `Backend` 已配置 |
| **`edit_file`** | 在文件内做字符串替换 | `Backend` 已配置 |
| **`glob`** | 按 glob 模式查找文件 | `Backend` 已配置 |
| **`grep`** | 在文件内容中按模式搜索（多输出模式） | `Backend` 已配置 |
| **`execute`** | 执行 Shell 命令（同步或流式输出） | 需额外提供 **`Shell` 或 `StreamingShell`**（二者互斥选其一） |

**说明：**

- **`execute`** 依赖 **`filesystem.Shell` / `StreamingShell`**。本机 **Local Backend** 在文档示例中可同时作为 **`Backend`** 与 **`StreamingShell`**（或同步执行侧）传入，从而打开命令执行类工具。
- 每个工具都可通过 `MiddlewareConfig` 里对应的 **`LsToolConfig`、`ReadFileToolConfig`、…** 做 **禁用（`Disable: true`）**、改描述、甚至 **`CustomTool` 替换实现**。

---

## 三、本机 Local Backend 行为要点（影响工具「实际能做什么」）

以下内容来自 Local Backend 文档，落地 Agent 时务必遵守：

1. **路径必须是绝对路径**（以 `/` 开头）。相对路径需先转换，例如：  
   `filepath.Abs("./relative/path")`。
2. **安全**：可在 `local.Config` 中配置 **`ValidateCommand`**，对 **`Execute` / 流式执行** 传入的命令做白名单或校验，避免模型误执行高危命令。
3. **Grep 工具链**：本机 **`grep` 类能力依赖系统是否安装 **ripgrep (`rg`)**；未安装时可能报错，需按官方 FAQ 安装。
4. **GrepRaw / 搜索语义**：文档示例中说明本地实现支持 **正则**（与 ripgrep 行为一致）；若你自定义 Backend，需以该实现为准。
5. **平台**：文档说明 **不支持 Windows**（依赖 `/bin/sh` 等），适用于 Unix/Linux/macOS 类环境。
6. **读文件分页**：Read 支持分页，文档提到默认例如 **200 行** 等与实现相关的默认行为，大文件应分页读取，避免一次灌满上下文。

Local Backend 提供的底层能力（与工具一一对应）包括：`LsInfo`、`Read`、`Write`、`Edit`、`GrepRaw`、`GlobInfo`、`Execute`、`ExecuteStreaming` 等，详见 [Local File System - API Reference](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/filesystem_backend/backend_%E6%9C%AC%E5%9C%B0%E6%96%87%E4%BB%B6%E7%B3%BB%E7%BB%9F/)。

---

## 四、基于本机 Local + FileSystem 开发「带 tools 的 Agent」的最佳实践

### 1. 推荐组合：`ChatModelAgent` + `filesystem` 中间件 + `ToolCallingChatModel`

- 使用 **`adk.NewChatModelAgent`**，在配置中挂载 **`Middlewares: []adk.ChatModelAgentMiddleware`**，将 **`filesystem.New(..., &filesystem.MiddlewareConfig{ Backend: localBackend, Shell/StreamingShell: ... })`** 放入其中。
- 模型使用 **`components/model.ToolCallingChatModel`**（`WithTools`），避免已废弃且并发不安全的 `BindTools`。
- 官方推荐使用 **`filesystem.New` + `MiddlewareConfig`**（而非旧版 `NewMiddleware` API），以便正确做 **BeforeAgent** 与上下文传播。

> 注意：个别文档页示例字段名可能写作 `Handlers`，以当前 **`ChatModelAgentConfig` 源码** 为准（一般为 **`Middlewares`**）。

### 2. 最小集成骨架（逻辑结构）

```text
local.NewBackend(ctx, &local.Config{ ValidateCommand: optional })
  → fsMiddleware.New(ctx, &fsMiddleware.MiddlewareConfig{
       Backend: backend,
       StreamingShell: backend, // 或 Shell: backend，按文档二选一
     })
  → adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
       Model: toolCallingModel,
       Middlewares: []adk.ChatModelAgentMiddleware{ fsMiddleware },
       // ToolsConfig: 其他业务工具可并列配置
     })
```

业务上若还有 **HTTP、搜索、MCP** 等工具，仍在 **`ToolsConfig`** 中与文件工具 **并列** 提供；文件类工具由中间件注入，无需再手写一遍 `read_file` 的 `BaseTool`（除非你用 `CustomTool` 覆盖）。

### 3. 安全与权限

- **生产或共享环境**：对 **`write_file` / `edit_file` / `execute`** 采用 **白名单目录**（通过 Instruction 约束 + **`ValidateCommand`** + 操作系统权限）组合治理；只读场景可对 **`WriteFileToolConfig` / `EditFileToolConfig` / execute 相关** 做 **`Disable: true`**。
- **绝对路径**：在系统提示或 Instruction 中明确要求模型只访问允许的路径前缀。

### 4. 成本与稳定性

- 为 **`ChatModelAgent` 设置合理 `MaxIterations`**，避免 ReAct 循环失控。
- **大结果**：若工具返回体积极大，关注官方对 **ToolReduction** 等中间件的迁移说明（旧版「大结果卸载」在 `MiddlewareConfig` 推荐路径中的变化以文档为准）。
- **流式命令**：长耗时任务优先 **`StreamingShell`**，避免阻塞与超时。

### 5. 国际化与可观测性

- 工具描述与系统提示支持中英文切换时可使用 **`adk.SetLanguage`**；或通过 **`ToolConfig.Desc` / `CustomSystemPrompt`** 精调。
- 需要审计时，结合 **`callbacks.Handler`** 记录工具调用前后信息（注意流式回调需 **关闭 `StreamReader`**）。

### 6. 与 DeepAgents 的关系

- **DeepAgents** 在配置 **`filesystem.Backend` / Shell** 时，会按产品逻辑挂载**同类文件/命令能力**；若你只用 **普通 ChatModelAgent**，通过 **同一套 `filesystem` 中间件 + Local Backend** 即可获得**同一批工具语义**，无需绑定 Deep 模式。

---

## 五、参考链接

- [FileSystem 中间件（工具列表与 MiddlewareConfig）](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/middleware_filesystem/)
- [FileSystem Backend 总览（InMemory / Local / Agentkit 对比）](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/filesystem_backend/)
- [Local Backend（安装、绝对路径、ValidateCommand、与 Agent 集成示例）](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/eino_adk_chatmodelagentmiddleware/filesystem_backend/backend_%E6%9C%AC%E5%9C%B0%E6%96%87%E4%BB%B6%E7%B3%BB%E7%BB%9F/)
- 同仓库梳理：[eino-interfaces.md](./eino-interfaces.md)、[eino-agents-and-practices.md](./eino-agents-and-practices.md)
