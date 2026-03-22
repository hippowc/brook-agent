from __future__ import annotations

from typing import Any

from openai import OpenAI

from brook_agent.session_log import SessionLogger


def create_client(api_key: str, base_url: str | None) -> OpenAI:
    kwargs: dict[str, Any] = {"api_key": api_key}
    if base_url:
        kwargs["base_url"] = base_url
    return OpenAI(**kwargs)


def chat_completion(
    client: OpenAI,
    model: str,
    messages: list[dict[str, Any]],
    *,
    session_logger: SessionLogger | None = None,
) -> str:
    resp = client.chat.completions.create(model=model, messages=messages)
    if session_logger is not None:
        session_logger.log_llm_round(
            model=model,
            messages=messages,
            tools=None,
            response=resp,
        )
    choice = resp.choices[0].message
    return (choice.content or "").strip()
