from __future__ import annotations

import json

from brook_agent.workspace import Workspace

TOOL = {
    "type": "function",
    "function": {
        "name": "write_file",
        "description": "在工作区内创建或覆盖文本文件（UTF-8）。会自动创建父目录。",
        "parameters": {
            "type": "object",
            "properties": {
                "path": {
                    "type": "string",
                    "description": "相对工作区的文件路径",
                },
                "content": {
                    "type": "string",
                    "description": "写入的完整文本内容",
                },
            },
            "required": ["path", "content"],
        },
    },
}


def run(ws: Workspace, arguments: str) -> str:
    args = json.loads(arguments or "{}")
    path = str(args.get("path", ""))
    content = args.get("content", "")
    if not isinstance(content, str):
        return "错误：content 必须是字符串。"

    target = ws.resolve(path)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(content, encoding="utf-8", newline="\n")
    return f"已写入：{path}（{len(content.encode('utf-8'))} 字节）"
