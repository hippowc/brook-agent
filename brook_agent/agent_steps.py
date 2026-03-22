from __future__ import annotations

from typing import Any, Callable

from agentloop import stop
from openai import OpenAI

from brook_agent.llm import chat_completion

StepFn = Callable[[Any, Any], Any]


def make_llm_pipeline_steps(
    client: OpenAI,
    model: str,
    user_message: str,
) -> list[StepFn]:
    """两轮 step：组装 messages → 调用 LLM，展示 agentloop 的 next_output 传递。"""

    def build_messages(_prev: Any, _loop_data: Any) -> list[dict[str, str]]:
        return [{"role": "user", "content": user_message}]

    def call_llm(messages: list[dict[str, str]], loop_data: Any) -> str:
        text = chat_completion(client, model, messages)
        print(text)
        stop(loop_data)
        return text

    return [build_messages, call_llm]
