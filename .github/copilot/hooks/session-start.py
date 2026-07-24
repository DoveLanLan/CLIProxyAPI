#!/usr/bin/env python3
"""
GitHub Copilot SessionStart hook for open-spec-code.

Outputs VS Code compatible hook JSON:
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "..."
  }
}
"""

from __future__ import annotations

import json
import os
import subprocess
import sys
from io import StringIO
from pathlib import Path

DEFAULT_TASK_SUMMARY_MAX_CHARS = 1000
DEFAULT_CONTEXT_MAX_CHARS = 6000


def should_skip() -> bool:
    return os.environ.get("COPILOT_NON_INTERACTIVE") == "1"


def find_repo_root(start: Path) -> Path:
    current = start
    while True:
        if (current / ".osc").is_dir() or (current / ".git").is_dir():
            return current
        if current.parent == current:
            return start
        current = current.parent


def read_text(path: Path, fallback: str = "", max_chars: int | None = None) -> str:
    try:
        content = path.read_text(encoding="utf-8")
    except (FileNotFoundError, PermissionError):
        return fallback

    if max_chars is not None and len(content) > max_chars:
        return content[:max_chars].rstrip() + "\n...[truncated]"
    return content


def env_int(name: str, default: int, minimum: int = 0) -> int:
    raw = os.environ.get(name, "").strip()
    if not raw:
        return default
    try:
        value = int(raw)
    except ValueError:
        return default
    return max(minimum, value)


def context_mode() -> str:
    mode = os.environ.get("OSC_CONTEXT_MODE", "compact").strip().lower()
    return "full" if mode == "full" else "compact"


def maybe_cap_context(text: str, mode: str) -> str:
    if mode == "full":
        return text
    max_chars = env_int("OSC_CONTEXT_MAX_CHARS", DEFAULT_CONTEXT_MAX_CHARS, 1000)
    if len(text) <= max_chars:
        return text
    return text[:max_chars].rstrip() + "\n...[truncated by OSC_CONTEXT_MAX_CHARS]\n"


def run_script(path: Path, cwd: Path) -> str:
    try:
        result = subprocess.run(
            [str(path)],
            cwd=str(cwd),
            capture_output=True,
            text=True,
            timeout=5,
            check=False,
        )
        output = (result.stdout or "").strip()
        return output if output else "No context available"
    except Exception:
        return "No context available"


def workflow_index(workflow_path: Path) -> str:
    content = read_text(workflow_path)
    if not content:
        return "No .osc/workflow.md found"

    lines = [
        "Workflow index (read the full file on demand: .osc/workflow.md)",
        "",
    ]
    for line in content.splitlines():
        if line.startswith("## "):
            lines.append(line)
    if len(lines) == 2:
        lines.append("(No section headings found)")
    return "\n".join(lines)


def resolve_current_task(repo_root: Path) -> tuple[str | None, Path | None]:
    task_ref = read_text(repo_root / ".osc" / ".current-task").strip()
    if not task_ref:
        return None, None

    task_path = Path(task_ref)
    if task_path.is_absolute():
        task_dir = task_path
    else:
        task_dir = repo_root / task_ref

    if not task_dir.is_dir():
        return task_ref, None
    return task_ref, task_dir


def change_artifacts_exist(task_dir: Path) -> bool:
    changes_dir = task_dir / "changes"
    return all((changes_dir / name).is_file() for name in ("proposal.md", "spec.md", "tasks.md"))


def summarize_task(task_ref: str | None, task_dir: Path | None, mode: str = "compact") -> str:
    if not task_ref:
        return (
            "Status: NO ACTIVE TASK\n"
            "Next: Ask what the user wants to work on. If it is a code change, "
            "create/select a task before implementation."
        )
    if task_dir is None:
        return (
            "Status: STALE POINTER\n"
            f"Task: {task_ref}\n"
            "Next: Fix `.osc/.current-task`, archive the stale task reference, or start a new task."
        )

    task_json = {}
    task_json_path = task_dir / "task.json"
    if task_json_path.is_file():
        try:
            task_json = json.loads(task_json_path.read_text(encoding="utf-8"))
        except (json.JSONDecodeError, PermissionError):
            task_json = {}

    task_name = task_json.get("name") or task_dir.name
    task_type = task_json.get("type") or "unknown"
    task_status = task_json.get("status") or "unknown"
    change_status = task_json.get("change_workflow_status") or "unknown"
    requires_change_workflow = bool(task_json.get("requires_change_workflow"))
    artifacts_ready = change_artifacts_exist(task_dir)
    prd_max = None if mode == "full" else env_int("OSC_TASK_SUMMARY_MAX_CHARS", DEFAULT_TASK_SUMMARY_MAX_CHARS, 100)
    prd = read_text(task_dir / "prd.md", "No PRD found", max_chars=prd_max)

    if task_status in {"done", "completed"}:
        archive_cmd = f"./.osc/scripts/task.sh archive {task_ref}"
        return (
            "Status: COMPLETED\n"
            f"Task: {task_name}\n"
            f"Next: Archive/close this task with `{archive_cmd}` or start/select a new task."
        )

    if not (task_dir / "prd.md").is_file():
        return (
            "Status: NOT READY\n"
            f"Task: {task_name}\n"
            "Missing: prd.md\n"
            "Next: Refine the task PRD before implementation."
        )

    if requires_change_workflow and not artifacts_ready:
        return (
            "Status: NOT READY\n"
            f"Task: {task_name}\n"
            f"Type: {task_type}\n"
            f"Change workflow: {change_status}\n"
            "Missing: task-level change artifacts (`proposal.md`, `spec.md`, `tasks.md`)\n"
            "Next: Run `change-workflow` and write artifacts under `.osc/tasks/<task>/changes/`."
        )

    return (
        "Status: READY\n"
        f"Task: {task_name}\n"
        f"Task ref: {task_ref}\n"
        f"Type: {task_type}\n"
        f"Task status: {task_status}\n"
        f"Change workflow: {change_status}\n\n"
        f"PRD summary (read full file on demand: {task_ref}/prd.md):\n{prd}"
    )


def read_guidelines(repo_root: Path) -> str:
    osc_dir = repo_root / ".osc"
    return "\n\n".join(
        [
            "**Note**: The files below are index files. Read the concrete guide files they reference before changing code.",
            "## Shared\n" + read_text(osc_dir / "spec" / "shared" / "index.md", "Not configured"),
            "## Frontend\n" + read_text(osc_dir / "spec" / "frontend" / "index.md", "Not configured"),
            "## Backend\n" + read_text(osc_dir / "spec" / "backend" / "index.md", "Not configured"),
            "## Guides\n" + read_text(osc_dir / "spec" / "guides" / "index.md", "Not configured"),
        ]
    )


def main() -> int:
    if should_skip():
        return 0

    try:
        hook_input = json.loads(sys.stdin.read() or "{}")
    except json.JSONDecodeError:
        hook_input = {}

    start_dir = Path(hook_input.get("cwd") or os.getcwd()).resolve()
    repo_root = find_repo_root(start_dir)
    osc_dir = repo_root / ".osc"
    mode = context_mode()

    task_ref, task_dir = resolve_current_task(repo_root)

    output = StringIO()
    output.write("<session-context>\n")
    output.write("You are starting a new session in an open-spec-code managed project.\n")
    output.write("This context came from the GitHub Copilot SessionStart hook.\n")
    output.write(f"Context mode: {mode}\n")
    output.write("</session-context>\n\n")

    output.write("<current-state>\n")
    output.write(run_script(osc_dir / "scripts" / "get-context.sh", repo_root))
    output.write("\n</current-state>\n\n")

    output.write("<current-task>\n")
    output.write(summarize_task(task_ref, task_dir, mode))
    output.write("\n</current-task>\n\n")

    output.write("<workflow>\n")
    if mode == "full":
        output.write(read_text(osc_dir / "workflow.md", "No .osc/workflow.md found"))
    else:
        output.write(workflow_index(osc_dir / "workflow.md"))
    output.write("\n</workflow>\n\n")

    output.write("<guidelines>\n")
    output.write(read_guidelines(repo_root))
    output.write("\n</guidelines>\n\n")

    output.write("<next-step>\n")
    output.write(
        "Context is already injected above. Do not re-read everything by default.\n"
        "If there is an active task, ask whether to continue it.\n"
        "If the user wants a guided workflow, the project includes Copilot prompt files under "
        "`.github/prompts/` such as `/start`, `/brainstorm`, `/before-dev`, `/check`, and `/finish-work`.\n"
        "For code changes, follow the active task's `.osc/tasks/<task>/changes/` artifacts as the implementation contract."
    )
    output.write("\n</next-step>")

    context = maybe_cap_context(output.getvalue(), mode)
    payload = {
        "suppressOutput": True,
        "systemMessage": f"OSC Copilot context injected ({len(context)} chars)",
        "hookSpecificOutput": {
            "hookEventName": "SessionStart",
            "additionalContext": context,
        },
    }
    print(json.dumps(payload, ensure_ascii=False), flush=True)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
