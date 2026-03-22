from __future__ import annotations

import json
from pathlib import Path

from brook_agent.workspace import Workspace

TOOL = {
    "type": "function",
    "function": {
        "name": "find_files",
        "description": (
            "在工作区内按目录与通配符列出文件路径（相对工作区）。"
            "不含「**」的模式会自动加上「**/」前缀，从而在起始目录下递归匹配（例如 *.md 等价于 **/*.md）。"
            "若模式里已含 **，则按 pathlib 的 glob 规则解析。"
        ),
        "parameters": {
            "type": "object",
            "properties": {
                "directory": {
                    "type": "string",
                    "description": "起始相对目录，默认 \".\"",
                    "default": ".",
                },
                "glob_pattern": {
                    "type": "string",
                    "description": (
                        "glob。已含 ** 时按 pathlib 规则解析；"
                        "否则自动加上 **/ 前缀以递归子目录（如 *.md → **/*.md）。"
                        "默认 \"**/*\"。"
                    ),
                    "default": "**/*",
                },
                "max_results": {
                    "type": "integer",
                    "description": "最多返回条数，默认 200",
                    "default": 200,
                },
            },
            "required": [],
        },
    },
}


def _effective_pattern(pattern: str) -> str:
    p = (pattern or "**/*").strip() or "**/*"
    if "**" in p:
        return p
    # pathlib.Path.glob("*.md") 仅匹配当前目录一层，常见意图是整树匹配
    return "**/" + p.lstrip("./\\")


def run(ws: Workspace, arguments: str) -> str:
    args = json.loads(arguments or "{}")
    directory = str(args.get("directory", ".") or ".")
    pattern = str(args.get("glob_pattern", "**/*") or "**/*")
    max_results = int(args.get("max_results", 200))
    if max_results < 1 or max_results > 2000:
        return "错误：max_results 须在 1～2000 之间。"

    base = ws.resolve(directory)
    if not base.is_dir():
        return f"错误：不是目录：{directory}"

    eff = _effective_pattern(pattern)
    paths: list[str] = []
    for p in sorted(base.glob(eff)):
        if not p.is_file():
            continue
        try:
            rel = p.resolve().relative_to(ws.root)
        except ValueError:
            continue
        paths.append(rel.as_posix())
        if len(paths) >= max_results:
            break

    if not paths:
        return "未找到匹配文件。"
    extra = "\n...(已达 max_results 上限)" if len(paths) >= max_results else ""
    return "\n".join(paths) + extra
