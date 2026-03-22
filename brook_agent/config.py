import os
from dataclasses import dataclass
from pathlib import Path


@dataclass(frozen=True)
class Settings:
    api_key: str
    base_url: str | None
    model: str
    user_message: str
    run_mode: str
    system_prompt: str | None
    workspace_root: str
    enable_file_tools: bool
    session_log_enabled: bool
    session_log_dir: str
    session_log_max_detail_chars: int


def load_settings() -> Settings:
    api_key = os.environ.get("OPENAI_API_KEY", "").strip()
    if not api_key:
        raise SystemExit("缺少 OPENAI_API_KEY，请在环境变量或 .env 中配置。")
    base = os.environ.get("OPENAI_BASE_URL", "").strip()
    mode = os.environ.get("BROOK_AGENT_MODE", "terminal").strip().lower()
    if mode not in ("terminal", "once"):
        mode = "terminal"
    sys_prompt = os.environ.get("SYSTEM_PROMPT", "").strip()
    raw_ws = os.environ.get("BROOK_AGENT_WORKSPACE", "").strip()
    workspace_root = str(
        Path(raw_ws).expanduser().resolve() if raw_ws else Path.cwd().resolve()
    )
    tools_raw = os.environ.get("BROOK_AGENT_FILE_TOOLS", "true").strip().lower()
    enable_file_tools = tools_raw not in ("0", "false", "no", "off")

    log_raw = os.environ.get("BROOK_AGENT_SESSION_LOG", "true").strip().lower()
    session_log_enabled = log_raw not in ("0", "false", "no", "off")
    log_dir = os.environ.get("BROOK_AGENT_LOG_DIR", "logs/sessions").strip()
    if not log_dir:
        log_dir = "logs/sessions"
    try:
        max_detail = int(os.environ.get("BROOK_AGENT_LOG_MAX_DETAIL_CHARS", "200000"))
    except ValueError:
        max_detail = 200_000
    if max_detail < 10_000:
        max_detail = 10_000

    return Settings(
        api_key=api_key,
        base_url=base or None,
        model=os.environ.get("LLM_MODEL", "gpt-4o-mini").strip(),
        user_message=os.environ.get(
            "USER_MESSAGE", "用一句话介绍你自己。"
        ).strip(),
        run_mode=mode,
        system_prompt=sys_prompt or None,
        workspace_root=workspace_root,
        enable_file_tools=enable_file_tools,
        session_log_enabled=session_log_enabled,
        session_log_dir=log_dir,
        session_log_max_detail_chars=max_detail,
    )
