---
description: "open-spec-code Copilot prompt: Brainstorm"
---

# Brainstorm

Use this before implementation when the task is vague, multi-file, or has design tradeoffs.

Rules:

- Create or select a task first.
- Update the task `prd.md` as soon as new facts are known.
- Ask one question at a time.
- Prefer concrete options with tradeoffs.
- Inspect the repo before asking the user for information that can be derived locally.

Workflow:

1. Create/select the task.
2. Seed or update `prd.md` with:
   - goal
   - known constraints
   - assumptions
   - open questions
   - acceptance criteria
   - out of scope
3. Inspect the repo and relevant docs first.
4. Ask only blocking or preference questions.
5. Once the requirements are stable, write:
   - `.osc/tasks/<task>/changes/proposal.md`
   - `.osc/tasks/<task>/changes/spec.md`
   - `.osc/tasks/<task>/changes/tasks.md`
6. Do not implement until the user confirms the converged scope when confirmation is needed.
