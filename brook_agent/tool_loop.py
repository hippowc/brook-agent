from __future__ import annotations

from typing import Any

from openai import OpenAI

from brook_agent.session_log import SessionLogger
from brook_agent.tools.registry import run_tool, tool_specs
from brook_agent.workspace import Workspace


def _append_assistant(messages: list[dict[str, Any]], msg: Any) -> None:
    entry: dict[str, Any] = {"role": "assistant", "content": msg.content}
    tool_calls = getattr(msg, "tool_calls", None)
    if tool_calls:
        entry["tool_calls"] = [
            {
                "id": tc.id,
                "type": getattr(tc, "type", None) or "function",
                "function": {
                    "name": tc.function.name,
                    "arguments": tc.function.arguments or "{}",
                },
            }
            for tc in tool_calls
        ]
        if not entry["content"]:
            entry["content"] = None
    messages.append(entry)


def run_turn_with_tools(
    client: OpenAI,
    model: str,
    messages: list[dict[str, Any]],
    workspace: Workspace,
    *,
    max_tool_rounds: int = 12,
    session_logger: SessionLogger | None = None,
) -> str:
    tools = tool_specs()
    for _ in range(max_tool_rounds):
        resp = client.chat.completions.create(
            model=model,
            messages=messages,
            tools=tools,
            tool_choice="auto",
        )
        if session_logger is not None:
            session_logger.log_llm_round(
                model=model,
                messages=list(messages),
                tools=tools,
                response=resp,
            )
        msg = resp.choices[0].message
        tool_calls = getattr(msg, "tool_calls", None)
        if not tool_calls:
            text = (msg.content or "").strip()
            messages.append({"role": "assistant", "content": text})
            return text

        _append_assistant(messages, msg)
        for tc in tool_calls:
            args = tc.function.arguments or "{}"
            result = run_tool(workspace, tc.function.name, args)
            if session_logger is not None:
                session_logger.log_tool_call(
                    tool_call_id=tc.id,
                    name=tc.function.name,
                    arguments=args,
                    result=result,
                )
            messages.append(
                {
                    "role": "tool",
                    "tool_call_id": tc.id,
                    "content": result,
                }
            )

    fallback = "已达到工具调用轮次上限，请缩小任务或分步说明。"
    messages.append({"role": "assistant", "content": fallback})
    return fallback
