# Context / Problem

Claude Code requests routed to an Antigravity OAuth account on bytevirt fail with HTTP 400 when a tool parameter is an array without an `items` schema. Antigravity rejects the translated request with `GenerateContentRequest...properties[cookies].items: missing field`.

# Goals

- Make Antigravity tool-schema cleanup produce upstream-valid array schemas when the client omits `items`.
- Preserve existing schema information and behavior for arrays that already define `items`.
- Add a focused regression test for the Claude Code failure shape.

# Constraints

- Keep the canonical translation architecture intact.
- Do not expose credentials or modify production state in repository artifacts.
- Avoid changes under the protected translator boundary.

# Non-goals

- Changing Antigravity authentication or model routing.
- Changing Claude Code settings or selecting a different upstream account.

# Proposed Approach

Extend the shared Gemini/Antigravity JSON-schema sanitizer to add a permissive object `items` schema for array nodes that lack one. Cover nested arrays and the exact `cookies` regression shape with unit tests.

# Risks & Mitigations

- A missing `items` schema does not identify the element type; use an object schema with permissive properties so tool calls remain valid and data is not discarded.
- Existing arrays with explicit `items` remain unchanged and are covered by existing tests.
