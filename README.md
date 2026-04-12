# Brook

基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) ADK 的可配置终端 Agent：通过 YAML 选择模型、工具、编排模式（ReAct、Deep、串行/并行/循环、Supervisor、Plan-Execute 等），提供 **`brook`（CLI 单次查询）** 与 **`brook-tui`（交互式终端 UI）**。

## 功能概览

- **配置驱动**：`~/.brook/agent.yaml`（首次运行自动生成），亦可指定 `--config` 指向任意路径。
- **多模式 Agent**：`react`、`deep`、`sequential`、`parallel`、`loop`、`supervisor`、`plan_execute`（说明见 [`doc/agent-configuration-guide.md`](doc/agent-configuration-guide.md)）。
- **工具**：本地文件系统（`read_file` / `glob` / `execute` 等，取决于配置）、可扩展中间件。
- **TUI**：多轮对话、`/help`、`/config`、`/agent mode`、`/new`、Tab 补全；会话存档于 `~/.brook/conversations/`。
- **工程细节**：工具调用失败时通过中间件转为模型可见的 observation（避免整轮 `NodeRunError` 直接中断）。

**要求**：Go **1.24+**（若从源码构建）；运行期需按配置提供 OpenAI 兼容 API、Ollama 等模型端点。

## 一键安装

### 从 GitHub Release 安装

```bash
curl -fsSL https://raw.githubusercontent.com/hippowc/brook/main/scripts/install.sh | bash
```

- 默认将 `brook` / `brook-tui` 安装到 `~/.local/bin` 或 `/usr/local/bin`（视权限而定）。
- 指定版本：`VERSION=v0.1.0 curl -fsSL ... | bash`
- 强制用 Go 从源码安装：`BROOK_FORCE_SOURCE=1 curl -fsSL ... | bash`

安装脚本会先请求 **GitHub API** 再下载 **Release 资源**。访问 GitHub 较慢时，下载可能持续较久；可配置代理，例如：`export HTTPS_PROXY=http://127.0.0.1:7890`（按你的代理修改）。若 Release 下载仍失败，脚本会回退到 **`go install`**（需本机已装 Go）。

### 使用 Go 安装（需已配置 `GOPATH/bin` 到 PATH）

```bash
go install github.com/hippowc/brook/cmd/brook@latest
go install github.com/hippowc/brook/cmd/brook-tui@latest
```

## 从源码构建

```bash
git clone https://github.com/hippowc/brook.git
cd brook
go build -o brook ./cmd/brook
go build -o brook-tui ./cmd/brook-tui
```

### 发布用交叉编译（macOS / Linux）

```bash
# 与 GitHub Release 标签一致，否则一键安装会 404（见下表）
VERSION=v0.0.1 ./scripts/build_release.sh
```

产物在 `dist/`：`brook_<VERSION>_<os>_<arch>.tar.gz` 与 `checksums.txt`。

**发布 Release 时附件名必须与标签一致。** 一键安装会请求例如：

| Release 标签 | 需上传的附件名（示例） |
|--------------|------------------------|
| `v0.0.1` | `brook_v0.0.1_darwin_amd64.tar.gz`、`brook_v0.0.1_darwin_arm64.tar.gz`、`brook_v0.0.1_linux_amd64.tar.gz`、`brook_v0.0.1_linux_arm64.tar.gz` |

若只运行 `./scripts/build_release.sh` 且未设 `VERSION`，会得到 `brook_4c53307_...` 这类名字，**与 `v0.0.1` Release 不匹配**，安装脚本会 404。请用上述带 `VERSION=...` 的命令重新打包并上传，或在 GitHub 网页上把附件**重命名**为表中形式。

## 快速使用

1. **首次配置**  
   运行任意命令会自动生成 `~/.brook/agent.yaml`。也可复制 [`config/agent.example.yaml`](config/agent.example.yaml) 后修改路径与 API。

2. **环境变量**  
   在 YAML 的 `models.providers.*.api_key_env` 中配置（如 `OPENAI_API_KEY`）。

3. **CLI 单次查询**

   ```bash
   brook -query "你好"
   ```

4. **TUI**

   ```bash
   brook-tui
   ```

5. **系统提示词**  
   支持多行 YAML（`|`）；或使用 `instruction: "@相对或绝对路径.md"` 引用 Markdown 文件（相对路径相对于 `agent.yaml` 所在目录）。

更完整的字段说明见 [`doc/agent-configuration-guide.md`](doc/agent-configuration-guide.md)。

## 仓库布局（简要）

| 路径 | 说明 |
|------|------|
| `cmd/brook` | 非交互 CLI |
| `cmd/brook-tui` | Bubble Tea TUI |
| `pkg/agentconfig` | YAML 模型与校验 |
| `internal/core/agent` | Agent 构建与工具错误中间件 |
| `config/agent.example.yaml` | 示例配置 |
