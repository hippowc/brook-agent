from __future__ import annotations

import json
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, TextIO


def _utc_now() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")


def _fence_text(body: str) -> str:
    fence = "```"
    while fence in body:
        fence += "`"
    return f"{fence}\n{body}\n{fence}"


def _truncate(s: str, limit: int) -> str:
    if limit <= 0 or len(s) <= limit:
        return s
    return s[:limit] + f"\n\n…（已截断，原文长度 {len(s)} 字符，上限 {limit}）"


def _json_detail(obj: Any, max_chars: int) -> str:
    try:
        if hasattr(obj, "model_dump"):
            raw = obj.model_dump(mode="json")
        else:
            raw = obj
        text = json.dumps(raw, ensure_ascii=False, indent=2, default=str)
    except Exception as exc:
        text = json.dumps({"_serialization_error": str(exc)}, ensure_ascii=False, indent=2)
    return _truncate(text, max_chars)


def _brief_messages(messages: list[dict[str, Any]]) -> str:
    n = len(messages)
    if not n:
        return "messages=0"
    last = messages[-1]
    role = last.get("role", "?")
    content = last.get("content")
    if isinstance(content, str):
        prev = content.replace("\n", " ")[:120]
        more = "…" if len(content) > 120 else ""
        tail = f"last={role!r} 预览「{prev}{more}」"
    else:
        tail = f"last={role!r} content={type(content).__name__}"
    has_tool = any(m.get("role") == "tool" for m in messages)
    tool_note = "；含 tool 回传" if has_tool else ""
    return f"messages={n}{tool_note}；{tail}"


def _brief_response(resp: Any) -> str:
    try:
        ch = resp.choices[0]
        fr = getattr(ch, "finish_reason", None)
        msg = ch.message
        tcs = getattr(msg, "tool_calls", None)
        if tcs:
            names = [getattr(getattr(tc, "function", None), "name", "?") for tc in tcs]
            return f"finish_reason={fr!r}；tool_calls={names}"
        c = (msg.content or "").replace("\n", " ")
        prev = c[:160] + ("…" if len(c) > 160 else "")
        return f"finish_reason={fr!r}；assistant 文本预览「{prev}」"
    except Exception as exc:
        return f"无法解析响应摘要：{exc}"


def _brief_tool(name: str, arguments: str) -> str:
    try:
        d = json.loads(arguments or "{}")
    except json.JSONDecodeError:
        return f"tool={name!r}；参数非 JSON，长度={len(arguments)}"
    parts = [f"{k}={repr(v)[:60]}" for k, v in list(d.items())[:5]]
    return f"tool={name!r}；" + "；".join(parts)


class SessionLogger:
    """按 Session → Request 分层；简要 / 详细分文件写入。"""

    def __init__(
        self,
        base_dir: Path,
        *,
        default_model: str,
        workspace_root: str | None,
        max_detail_chars: int = 200_000,
    ) -> None:
        self.session_id = str(uuid.uuid4())
        self.started_at = _utc_now()
        self.default_model = default_model
        self.workspace_root = workspace_root
        self.max_detail_chars = max_detail_chars

        self._session_dir = (base_dir / self.session_id).resolve()
        self._session_dir.mkdir(parents=True, exist_ok=True)
        self._brief_path = self._session_dir / "session_brief.md"
        self._detail_path = self._session_dir / "session_detail.md"
        self._fh_brief: TextIO = self._brief_path.open("w", encoding="utf-8")
        self._fh_detail: TextIO = self._detail_path.open("w", encoding="utf-8")

        self._request_idx = 0
        self._llm_round_in_req = 0
        self._tool_idx_in_req = 0

        self._write_headers()

    @property
    def brief_path(self) -> Path:
        return self._brief_path

    @property
    def detail_path(self) -> Path:
        return self._detail_path

    @property
    def log_path(self) -> Path:
        """兼容旧逻辑：默认指向简要日志路径。"""
        return self._brief_path

    def _write_brief(self, s: str) -> None:
        self._fh_brief.write(s)
        self._fh_brief.flush()

    def _write_detail(self, s: str) -> None:
        self._fh_detail.write(s)
        self._fh_detail.flush()

    def _write_headers(self) -> None:
        ws = self.workspace_root or "（未启用文件工具，无工作区约束）"
        table = (
            "| 字段 | 值 |\n"
            "|------|-----|\n"
            f"| Session ID | `{self.session_id}` |\n"
            f"| 开始 (UTC) | `{self.started_at}` |\n"
            f"| 默认模型 | `{self.default_model}` |\n"
            f"| 工作区根 | `{ws}` |\n"
        )
        self._write_brief(
            "# Brook Agent 会话日志 · **简要**\n\n"
            f"详细 JSON / 全文见同目录 **`session_detail.md`**。\n\n"
            "## Session 概览\n\n"
            f"{table}\n"
            "---\n\n"
        )
        self._write_detail(
            "# Brook Agent 会话日志 · **详细**\n\n"
            f"摘要脉络见 **`session_brief.md`**。单块文本上限 {self.max_detail_chars} 字符（超出截断）。\n\n"
            "## Session 概览\n\n"
            f"{table}\n"
            f"| 详细块上限 | {self.max_detail_chars} 字符 |\n\n"
            "---\n\n"
        )

    def start_request(self, user_text: str) -> None:
        self._request_idx += 1
        self._llm_round_in_req = 0
        self._tool_idx_in_req = 0
        summary = user_text.replace("\n", " ")[:200]
        more = "…" if len(user_text) > 200 else ""
        self._write_brief(
            f"## Request #{self._request_idx}\n\n"
            "### 用户输入（简要）\n\n"
            f"- 字符数：{len(user_text)}\n"
            f"- 摘要：`{summary}{more}`\n\n"
            "---\n\n"
        )
        self._write_detail(
            f"## Request #{self._request_idx}\n\n"
            "### 用户输入（详细 · 全文）\n\n"
            f"{_fence_text(user_text)}\n\n"
            "---\n\n"
        )

    def log_llm_round(
        self,
        *,
        model: str,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]] | None,
        response: Any,
    ) -> None:
        self._llm_round_in_req += 1
        tools_note = f"{len(tools)} 个" if tools else "无"
        self._write_brief(
            f"### LLM 调用 · Request #{self._request_idx} · 第 {self._llm_round_in_req} 轮\n\n"
            "#### 入参（简要）\n\n"
            f"- **模型：** `{model}`\n"
            f"- **{_brief_messages(messages)}**\n"
            f"- **tools：** {tools_note}\n\n"
            "#### 出参（简要）\n\n"
            f"- {_brief_response(response)}\n\n"
            "---\n\n"
        )
        self._write_detail(
            f"### LLM 调用 · Request #{self._request_idx} · 第 {self._llm_round_in_req} 轮\n\n"
            "#### 请求 · messages\n\n"
            f"{_fence_text(_json_detail(messages, self.max_detail_chars))}\n\n"
        )
        if tools:
            self._write_detail(
                "#### 请求 · tools 定义\n\n"
                f"{_fence_text(_json_detail(tools, self.max_detail_chars))}\n\n"
            )
        self._write_detail(
            "#### 响应 · API 返回体\n\n"
            f"{_fence_text(_json_detail(response, self.max_detail_chars))}\n\n"
            "---\n\n"
        )

    def log_tool_call(
        self,
        *,
        tool_call_id: str,
        name: str,
        arguments: str,
        result: str,
    ) -> None:
        self._tool_idx_in_req += 1
        res_prev = result.replace("\n", " ")[:160]
        res_more = "…" if len(result) > 160 else ""
        self._write_brief(
            f"### 工具调用 · Request #{self._request_idx} · 第 {self._tool_idx_in_req} 次\n\n"
            f"- **tool_call_id：** `{tool_call_id}`\n"
            f"- **name：** `{name}`\n"
            f"- **入参摘要：** {_brief_tool(name, arguments)}\n"
            f"- **出参长度：** {len(result)} 字符；预览「{res_prev}{res_more}」\n\n"
            "---\n\n"
        )
        self._write_detail(
            f"### 工具调用 · Request #{self._request_idx} · 第 {self._tool_idx_in_req} 次\n\n"
            f"- **tool_call_id：** `{tool_call_id}` · **name：** `{name}`\n\n"
            "#### 入参（原始 arguments）\n\n"
            f"{_fence_text(arguments)}\n\n"
            "#### 出参（完整文本）\n\n"
            f"{_fence_text(_truncate(result, self.max_detail_chars))}\n\n"
            "---\n\n"
        )

    def log_error(self, exc: BaseException) -> None:
        self._write_brief(
            f"### 异常 · Request #{self._request_idx}\n\n"
            f"- `{type(exc).__name__}`: {exc}\n\n"
            "---\n\n"
        )
        self._write_detail(
            f"### 异常 · Request #{self._request_idx}\n\n"
            f"{_fence_text(_truncate(str(exc), self.max_detail_chars))}\n\n"
            "---\n\n"
        )

    def close(self) -> None:
        end = f"\n---\n\n## Session 结束\n\n- **UTC：** `{_utc_now()}`\n"
        if not self._fh_brief.closed:
            self._write_brief(end)
            self._fh_brief.close()
        if not self._fh_detail.closed:
            self._write_detail(end)
            self._fh_detail.close()
