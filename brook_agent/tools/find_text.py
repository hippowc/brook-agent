from __future__ import annotations

import fnmatch
import json
from pathlib import Path

from brook_agent.workspace import Workspace

TOOL = {
    "type": "function",
    "function": {
        "name": "find_text",
        "description": "在工作区内按「子串」搜索文本（非正则）。可指定单文件或目录；目录下按 file_glob 过滤。",
        "parameters": {
            "type": "object",
            "properties": {
                "pattern": {
                    "type": "string",
                    "description": "要搜索的子串（区分大小写）",
                },
                "path": {
                    "type": "string",
                    "description": "相对工作区的文件或目录，默认 \".\"",
                    "default": ".",
                },
                "file_glob": {
                    "type": "string",
                    "description": "在目录内只匹配此类文件，如 \"*.py\"，默认 \"*\"",
                    "default": "*",
                },
                "max_matches": {
                    "type": "integer",
                    "description": "最多返回的匹配条数（行级），默认 80",
                    "default": 80,
                },
            },
            "required": ["pattern"],
        },
    },
}


def _iter_files(ws: Workspace, rel: str, file_glob: str) -> list[Path]:
    target = ws.resolve(rel)
    if target.is_file():
        return [target]
    if not target.is_dir():
        return []

    out: list[Path] = []
    for p in sorted(target.rglob("*")):
        if not p.is_file():
            continue
        try:
            p.relative_to(ws.root)
        except ValueError:
            continue
        if not fnmatch.fnmatch(p.name, file_glob):
            continue
        out.append(p)
    return out


def run(ws: Workspace, arguments: str) -> str:
    args = json.loads(arguments or "{}")
    pattern = str(args.get("pattern", ""))
    if not pattern:
        return "错误：pattern 不能为空。"

    rel = str(args.get("path", ".") or ".")
    file_glob = str(args.get("file_glob", "*") or "*")
    max_matches = int(args.get("max_matches", 80))
    if max_matches < 1 or max_matches > 500:
        return "错误：max_matches 须在 1～500 之间。"

    files = _iter_files(ws, rel, file_glob)
    if not files:
        return f"错误：路径不是可读文件/目录：{rel}"

    lines_out: list[str] = []
    capped = False
    for fp in files:
        try:
            text = fp.read_text(encoding="utf-8", errors="replace")
        except OSError as exc:
            lines_out.append(f"{_rel(ws, fp)}: [读取失败 {exc}]")
            continue
        for i, line in enumerate(text.splitlines(), start=1):
            if pattern not in line:
                continue
            rel_p = _rel(ws, fp)
            lines_out.append(f"{rel_p}:{i}:{line}")
            if len(lines_out) >= max_matches:
                capped = True
                break
        if capped:
            break

    if not lines_out:
        return "未找到匹配文本。"
    suffix = "\n...(已达 max_matches 上限)" if capped else ""
    return "\n".join(lines_out) + suffix


def _rel(ws: Workspace, p: Path) -> str:
    return p.resolve().relative_to(ws.root).as_posix()
