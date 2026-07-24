---
description: "open-spec-code Copilot prompt: Before Dev"
---

# Before Dev

Compatibility prompt for pre-development context. Use `project-spec` and `change-workflow` as the core skills.

1. Confirm `.osc/.current-task` points at the task you are about to work on.
2. Run:
   ```bash
   ./.osc/scripts/get-context.sh
   git diff --name-only HEAD
   ```
3. Always read:
   - `.osc/spec/shared/index.md`
   - `.osc/spec/guides/index.md`
4. Then read the relevant development indexes:
   - frontend work -> `.osc/spec/frontend/index.md`
   - backend work -> `.osc/spec/backend/index.md`
   - cross-layer work -> both
5. Do not stop at the indexes. Read the concrete guide files they reference.
6. If repo rules are missing or stale, run `project-spec` first.
7. If the task changes behavior, config, or APIs, run `change-workflow` and confirm task-level change artifacts already exist before editing.
