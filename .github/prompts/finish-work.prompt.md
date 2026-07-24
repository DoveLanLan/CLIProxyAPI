---
description: "open-spec-code Copilot prompt: Finish Work"
---

# Finish Work

Compatibility prompt for task closure. Use `change-workflow` artifacts and `quality-gate`.

1. Confirm the current task.
2. Write or update:
   - `change-summary.md`
   - `regression-checklist.md`
   - `rollback-notes.md`
3. Run `quality-gate`, preferring `./osc gate`, and update `.osc/quality-gate.md`.
4. If the task is complete, mark it done and archive it when appropriate:
   ```bash
   ./.osc/scripts/task.sh done --archive
   ```
5. Record the final state:
   - what changed
   - what was verified
   - what remains open

Do not claim completion if the artifacts or quality gate are missing.
