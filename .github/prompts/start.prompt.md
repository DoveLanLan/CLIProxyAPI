---
description: "open-spec-code Copilot prompt: Start Session"
---

# Start Session

Initialize the session before writing code.

1. Read `.osc/workflow.md`.
2. Run:
   ```bash
   ./.osc/scripts/get-context.sh
   ```
3. Read:
   - `.osc/spec/shared/index.md`
   - `.osc/spec/guides/index.md`
   - `.osc/spec/frontend/index.md` and/or `.osc/spec/backend/index.md`
4. If `.osc/.current-task` exists, read:
   - `task.json`
   - `prd.md`
   - `.osc/tasks/<task>/changes/proposal.md`
   - `.osc/tasks/<task>/changes/spec.md`
   - `.osc/tasks/<task>/changes/tasks.md`
5. Classify the user request:
   - Question: answer directly
   - Trivial fix: small direct edit, then run `quality-gate`
   - Simple task: create/select task, ensure change artifacts exist, then implement
   - Complex task: switch to `/brainstorm` first
6. If code will change, do not start editing until the current task and its change artifacts are ready.

After loading context, report what you found and ask whether to continue the active task if one exists.
