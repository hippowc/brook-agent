# Brook Agent 配置指南

本文说明 `agent.yaml`（或 `config/agent.example.yaml`）中**可配置项**及 **`agent.mode` 各模式如何写**。字段定义以源码为准：`pkg/agentconfig/types.go`、`pkg/agentconfig/validate.go`。

---

## 如何自行发现可配项

| 途径 | 说明 |
|------|------|
| **本文件** | 模式与常用块说明 |
| **`pkg/agentconfig/types.go`** | 全部结构体与 YAML 字段名、注释 |
| **`config/agent.example.yaml`** | 与实现对齐的完整示例 |
| **TUI** | 输入 `/help` 查看内置命令与模式速查；`/config` 直接编辑当前加载的 YAML |
| **默认首次生成** | `internal/brookdir/default_agent.yaml` 嵌入二进制，仅含最小可运行项；首次运行写入 `~/.brook/agent.yaml` |

---

## `agent.mode` 一览（Brook 已接线的值）

以下模式在 `internal/core/agent/builder.go` 中实现；`custom` **未接线**，加载会报错。

| `mode` 值 | 含义 | `agent.mode_config` 要求 |
|-----------|------|---------------------------|
| **`react`** | 单 Agent，内置 ReAct（`ChatModelAgent`） | 可为 `null`；无需子 Agent |
| **`deep`** | DeepAgents（规划 + Task + 子 Agent 等） | 可选 `deep` 子块；可选 `sub_agent_names` 提供额外子 Agent |
| **`sequential`** | 子 Agent **顺序**执行 | **必须** `sub_agent_names`（至少 1 个名称） |
| **`parallel`** | 子 Agent **并行**执行 | **必须** `sub_agent_names` |
| **`loop`** | 子 Agent 循环流水线 | **必须** `sub_agent_names`；可选 `loop_max_iterations` |
| **`supervisor`** | Supervisor + 工人子 Agent | **必须** `supervisor.supervisor_agent` + `sub_agent_names`（工人，可与 supervisor 名区分） |
| **`plan_execute`** | Planner / Executor / Replanner | **必须** `plan_execute.planner` / `executor` / `replanner`（三个**逻辑名**；Brook 会据此各建一个子 Agent，**不需要**再写 `sub_agent_names`） |
| **`custom`** | 自定义 | Brook **未实现**，请勿使用 |

校验逻辑见 `pkg/agentconfig/validate.go` 中 `validateMode()`。

### 子 Agent 名称（`sub_agent_names` 等）

Brook 用**同一套模型与 instruction 模板**按名称实例化多个 `ChatModelAgent`（见 `buildNamedAgents`）。名称仅用于区分角色，**不是**独立配置文件；复杂图结构仍需改代码扩展。

### `mode_config` 示例片段

**Deep（可选子块）：**

```yaml
agent:
  mode: deep
  mode_config:
    deep:
      without_write_todos: false
      without_general_sub_agent: false
      max_iteration: 0   # 0 表示用全局 max_iterations
    # sub_agent_names: ["researcher", "coder"]  # 可选
```

**Sequential / Parallel / Loop：**

```yaml
agent:
  mode: sequential
  mode_config:
    sub_agent_names: ["step-a", "step-b"]
    loop_max_iterations: 5   # 仅 loop 模式有意义
```

**Supervisor：**

```yaml
agent:
  mode: supervisor
  mode_config:
    supervisor:
      supervisor_agent: "lead"
    sub_agent_names: ["lead", "worker1", "worker2"]  # lead 与 supervisor_agent 一致，其余为工人
```

**Plan-Execute：**

```yaml
agent:
  mode: plan_execute
  mode_config:
    plan_execute:
      planner: "planner"
      executor: "executor"
      replanner: "replanner"
```

`buildPlanExecute` 会用这三个名字调用 `buildNamedAgents`，各实例化一个子 Agent，**不要求** `sub_agent_names`。

---

## 其它常用块（默认文件里可能未全部展开）

| 区块 | 作用 |
|------|------|
| **`agent.instruction` / `user_prompt`** | 支持整段 **`@路径`** 引用文件（相对 `agent.yaml` 目录或绝对路径），见 `pkg/agentconfig/atfile.go` |
| **`agent.working_directory`** | 本地工具工作目录，**须绝对路径** |
| **`agent.tools.filesystem`** | 本地/内存文件系统工具；`local.strict_commands: true` 时 shell 仅白名单命令（见 `internal/core/fs/backend.go`） |
| **`models`** | `providers` + `active`；密钥用 `api_key_env` |
| **`memory`** | Session 落盘、`output_key`、`max_context_messages`（TUI 裁剪上下文条数） |
| **`interrupt`** | Checkpoint 与 `brook-tui` / CLI 恢复相关 |
| **`a2ui`** | JSONL 流式 UI 输出 |

---

## 默认生成配置是否「够全」

首次运行写入的 **`~/.brook/agent.yaml`** 来自嵌入的 **`internal/brookdir/default_agent.yaml`**，设计目标是：**最小可运行**（`react` + 单模型 + filesystem + memory + interrupt），避免首次用户被长 YAML 淹没。

**更全的示例**请用仓库内 **`config/agent.example.yaml`** 或复制其中段落到 `~/.brook/agent.yaml`。

---

## 运行时切换模式（TUI）

- 命令：`/agent mode <模式名>`（与上表 `mode` 值一致；`custom` 不支持切换）
- 会**写回**当前加载的配置文件：除 `agent.mode` 外，会**按目标模式写入默认 `mode_config`（占位子 Agent 名等）**，并覆盖原有的 `mode_config`；成功后会 `Load` 并提示如何自行修改。
- 默认占位逻辑见源码 `pkg/agentconfig/mode_defaults.go`（`DefaultModeConfig`）。

---

## `gateway`（`brook-gateway`）

由 **`brook-gateway`** 读取与 `brook` / `brook-tui` **相同的** `agent.yaml`。将 **`gateway.enabled`** 设为 **`true`** 后启动进程即可监听 HTTP。

| 能力 | 说明 |
|------|------|
| 路由 | `GET /health`、`GET /ready`、`POST /v1/chat` |
| 请求体 | JSON：`text`（必填）、`user_id`（必填）、`conversation_id`（可选）；响应 `{"reply":"..."}` |
| 会话 | 按 `user_id` + `conversation_id` 派生键，**独立** 存 ADK `SessionValues`（与 `memory.session_file_path` 的 CLI/TUI 会话文件无关）；`session.store` 为 `memory` 或 `file`，`file` 默认目录 `~/.brook/gateway/sessions/` |
| 鉴权 | `auth.mode`：`none` \| `bearer`（`Authorization: Bearer <token>`，密钥来自 `bearer_token_env`）\| `hmac`（`X-Brook-Timestamp` Unix 秒 + `X-Brook-Signature` 为 hex(`HMAC-SHA256(secret, timestamp + "\\n" + raw_body)`)，密钥来自 `hmac_secret_env`） |
| 限流 | `rate_limit.enabled` 时按客户端 IP（支持 `X-Forwarded-For` / `X-Real-IP`）滑动窗口 |
| 并发 | 进程内对 **`Runner.Query` 串行化**（互斥锁）；水平扩展请多实例 + 前置负载均衡，会话需共享存储（如 `session.store: file` 指向共享盘） |

超时与体大小：`query_timeout_seconds`、`max_request_body_bytes`、`read_*` / `write_timeout_seconds` 等见 `pkg/agentconfig/types.go`。

---

## 延伸阅读

- `req/agent-config.md`：设计目标与字段映射摘要  
- `doc/eino-agents-and-practices.md`：Eino Agent 类型与概念  
