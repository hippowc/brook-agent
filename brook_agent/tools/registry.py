from __future__ import annotations

import json
from collections.abc import Callable

from brook_agent.tools import find_files, find_text, read_file, write_file
from brook_agent.workspace import Workspace

_TOOLS: list[dict] = [
    read_file.TOOL,
    write_file.TOOL,
    find_files.TOOL,
    find_text.TOOL,
]

_RUNNERS: dict[str, Callable[[Workspace, str], str]] = {
    "read_file": read_file.run,
    "write_file": write_file.run,
    "find_files": find_files.run,
    "find_text": find_text.run,
}


def tool_specs() -> list[dict]:
    return list(_TOOLS)


def run_tool(ws: Workspace, name: str, arguments: str) -> str:
    fn = _RUNNERS.get(name)
    if fn is None:
        return f"错误：未知工具 {name!r}。"
    try:
        return fn(ws, arguments)
    except json.JSONDecodeError as exc:
        return f"错误：工具参数不是合法 JSON：{exc}"
    except Exception as exc:
        return f"错误：{exc}"
