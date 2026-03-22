from __future__ import annotations

import json

from brook_agent.workspace import Workspace

TOOL = {
    "type": "function",
    "function": {
        "name": "read_file",
        "description": "读取工作区内单个文本文件内容（UTF-8，非法字节会替换）。大文件会被截断。",
        "parameters": {
            "type": "object",
            "properties": {
                "path": {
                    "type": "string",
                    "description": "相对工作区的文件路径",
                },
                "max_bytes": {
                    "type": "integer",
                    "description": "最多读取的字节数，默认 262144",
                    "default": 262144,
                },
            },
            "required": ["path"],
        },
    },
}


def run(ws: Workspace, arguments: str) -> str:
    args = json.loads(arguments or "{}")
    path = str(args.get("path", ""))
    max_bytes = int(args.get("max_bytes", 262144))
    if max_bytes < 1 or max_bytes > 2_097_152:
        return "错误：max_bytes 须在 1～2097152 之间。"

    target = ws.resolve(path)
    if not target.is_file():
        return f"错误：不是文件或不存在：{path}"

    data = target.read_bytes()[:max_bytes]
    try:
        text = data.decode("utf-8")
    except UnicodeDecodeError:
        text = data.decode("utf-8", errors="replace")

    truncated = target.stat().st_size > len(data)
    note = f"\n\n[已截断，共读取 {len(data)} 字节]" if truncated else ""
    return text + note
