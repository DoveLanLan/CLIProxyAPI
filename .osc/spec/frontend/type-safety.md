# Type Safety

> Type-safety patterns for UI-related code in this Go repository.

---

## Overview

There is no TypeScript layer in-tree. Type safety is provided by:

- Go structs and interfaces
- typed Bubble Tea messages and model fields
- YAML/JSON binding into concrete config/request structs
- validation on the backend before persistence

---

## Type Organization

- Keep long-lived UI state on explicit model structs.
- Prefer typed request/response structs in management code where the shape is stable.
- Use `map[string]any` only at genuine boundary layers where the payload is intentionally flexible.

---

## Validation

- Config writes should be reloaded and validated before becoming authoritative.
- JSON request bodies on management endpoints should use typed request structs or explicit bind checks.
- TUI/client code should treat backend responses as untrusted until parsed or checked.

Primary evidence: `internal/api/handlers/management/config_basic.go`, `internal/api/handlers/management/handler.go`

---

## Common Patterns

- typed message structs for Bubble Tea result handling
- explicit struct fields for tab state
- concrete config structs instead of loose maps in server code

---

## Forbidden Patterns

- Introducing a fake frontend type system vocabulary that does not match Go code
- Spreading `map[string]any` through the codebase when a stable struct would be clearer
- Persisting unvalidated UI-driven config changes directly to disk
