from __future__ import annotations

from pathlib import Path


class Workspace:
    """将 LLM 可见路径限制在单一根目录下，防止任意读盘。"""

    def __init__(self, root: Path) -> None:
        self.root = root.resolve()

    def resolve(self, relative_path: str) -> Path:
        rel = (relative_path or ".").strip().replace("\\", "/")
        if rel.startswith("/") or Path(rel).is_absolute():
            raise ValueError("仅允许相对工作区的路径，不能使用绝对路径。")
        candidate = (self.root / rel).resolve()
        try:
            candidate.relative_to(self.root)
        except ValueError as exc:
            raise ValueError("路径超出工作区范围。") from exc
        return candidate
