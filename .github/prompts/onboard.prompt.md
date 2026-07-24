---
description: "open-spec-code Copilot prompt: Onboard"
---

# Onboard

Audit whether this repository is ready for open-spec-code workflow execution.

1. Confirm `.osc/`, `.claude/`, `.codex/`, and `.github/` exist.
2. Read `.osc/workflow.md`.
3. Run:
   ```bash
   ./.osc/scripts/get-context.sh
   ```
4. Read the spec indexes under `.osc/spec/`.
5. Verify that task scripts are present under `.osc/scripts/`.
6. Report:
   - current developer
   - current task
   - missing workflow assets
   - what the user should run next

If anything critical is missing, tell the user exactly which command should restore it.
