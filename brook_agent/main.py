from __future__ import annotations

import sys
from pathlib import Path

from dotenv import load_dotenv

from agentloop import start

from brook_agent.agent_steps import make_llm_pipeline_steps
from brook_agent.config import load_settings
from brook_agent.llm import create_client
from brook_agent.session_log import SessionLogger
from brook_agent.terminal_chat import run_interactive_chat
from brook_agent.workspace import Workspace


def main() -> None:
    if hasattr(sys.stdout, "reconfigure"):
        try:
            sys.stdout.reconfigure(encoding="utf-8", errors="replace")
        except (OSError, ValueError):
            pass
    if hasattr(sys.stdin, "reconfigure"):
        try:
            sys.stdin.reconfigure(encoding="utf-8", errors="replace")
        except (OSError, ValueError):
            pass
    load_dotenv()
    settings = load_settings()
    client = create_client(settings.api_key, settings.base_url)

    session_logger: SessionLogger | None = None
    try:
        if settings.run_mode == "terminal" and settings.session_log_enabled:
            log_base = Path(settings.session_log_dir).expanduser().resolve()
            session_logger = SessionLogger(
                log_base,
                default_model=settings.model,
                workspace_root=settings.workspace_root
                if settings.enable_file_tools
                else None,
                max_detail_chars=settings.session_log_max_detail_chars,
            )

        if settings.run_mode == "terminal":
            ws: Workspace | None = None
            if settings.enable_file_tools:
                ws = Workspace(Path(settings.workspace_root))
            run_interactive_chat(
                client,
                settings.model,
                system_prompt=settings.system_prompt,
                workspace=ws,
                session_logger=session_logger,
            )
            return

        steps = make_llm_pipeline_steps(client, settings.model, settings.user_message)
        loop_data = start(steps, paused=False)
        loop_data["thread"].join()
    finally:
        if session_logger is not None:
            session_logger.close()


if __name__ == "__main__":
    main()
