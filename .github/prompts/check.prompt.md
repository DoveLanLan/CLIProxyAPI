---
description: "open-spec-code Copilot prompt: Check"
---

# Check

Compatibility prompt for post-change validation. Use `quality-gate` as the core skill.

1. Identify changed files:
   ```bash
   git diff --name-only HEAD
   ```
2. Read the relevant spec indexes again and the concrete guide files they reference.
3. Review the changed code against those rules.
4. Run the repository quality gate that applies to this change. Prefer:
   ```bash
   ./osc gate
   ```
   If `./osc gate` is unavailable, use the repo command selected from evidence. Default fallback:
   ```bash
   npm test
   ```
5. Persist the result to `.osc/quality-gate.md`.
6. Report:
   - findings or "no findings"
   - commands run
   - remaining risk if any checks could not be executed
