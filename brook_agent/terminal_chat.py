from __future__ import annotations

import copy
from typing import Any

from openai import OpenAI

from brook_agent.llm import chat_completion
from brook_agent.session_log import SessionLogger
from brook_agent.tool_loop import run_turn_with_tools
from brook_agent.workspace import Workspace

_EXIT = frozenset({"exit", "quit", "q", "/exit", "/quit"})


def _build_system_text(
    system_prompt: str | None,
    workspace: Workspace | None,
) -> str | None:
    parts: list[str] = []
    if system_prompt:
        parts.append(system_prompt.strip())
    if workspace is not None:
        parts.append(
            "你可以在「工作区」内使用工具：read_file、write_file、find_files、find_text。"
            "所有路径均为相对工作区的相对路径，禁止使用绝对路径或跳出工作区。"
            f"\n工作区根目录：{workspace.root}"
        )
    if not parts:
        return None
    return "\n\n".join(parts)


def run_interactive_chat(
    client: OpenAI,
    model: str,
    *,
    system_prompt: str | None = None,
    workspace: Workspace | None = None,
    session_logger: SessionLogger | None = None,
) -> None:
    messages: list[dict[str, Any]] = []
    combined = _build_system_text(system_prompt, workspace)
    if combined:
        messages.append({"role": "system", "content": combined})

    print("多轮对话已启动。输入 exit、quit 或 q 结束；Ctrl+C 或 Ctrl+Z+Enter 也可退出。\n")
    if session_logger is not None:
        print(
            "[会话日志]\n"
            f"  简要：{session_logger.brief_path}\n"
            f"  详细：{session_logger.detail_path}\n"
        )

    while True:
        try:
            line = input("你: ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\n再见。")
            break

        if not line:
            continue
        if line.lower() in _EXIT:
            print("再见。")
            break

        snapshot = copy.deepcopy(messages)
        if session_logger is not None:
            session_logger.start_request(line)
        messages.append({"role": "user", "content": line})
        try:
            if workspace is not None:
                reply = run_turn_with_tools(
                    client,
                    model,
                    messages,
                    workspace,
                    session_logger=session_logger,
                )
            else:
                reply = chat_completion(
                    client,
                    model,
                    messages,
                    session_logger=session_logger,
                )
                messages.append({"role": "assistant", "content": reply})
        except Exception as exc:
            messages.clear()
            messages.extend(snapshot)
            if session_logger is not None:
                session_logger.log_error(exc)
            print(f"[请求失败] {exc}\n")
            continue

        print(f"助手: {reply}\n")
