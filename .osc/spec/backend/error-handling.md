# Error Handling

> How errors are represented and propagated in CLIProxyAPI.

---

## Overview

This repository has two distinct error surfaces:

1. Public AI-compatible endpoints under `/v1` and `/v1beta`
2. Management endpoints under `/v0/management`

Do not treat them as interchangeable. Public routes must preserve client compatibility. Management routes prioritize operator clarity.

---

## Error Types

### Public API errors

Public OpenAI-compatible handlers use `sdk/api/handlers.ErrorResponse` and `BuildErrorResponseBody`.

Key behaviors:

- preserve upstream payloads when the upstream already returned valid JSON
- otherwise build a structured error object with `message`, `type`, and optional `code`
- map HTTP status to client-facing error categories such as `authentication_error`, `permission_error`, `rate_limit_error`, or `server_error`

Primary evidence: `sdk/api/handlers/handlers.go`

### Management API errors

Management handlers mostly return simple `gin.H` payloads such as:

- `{"error": "..."}`
- `{"error": "...", "message": "..."}`
- `{"status": "error", "error": "..."}`

This surface is intentionally more direct and operational.

Primary evidence: `internal/api/handlers/management/config_basic.go`, `internal/api/handlers/management/auth_files.go`, `internal/api/handlers/management/logs.go`, `internal/api/handlers/management/usage.go`

---

## Error Handling Patterns

- Return early on invalid input, failed auth, and failed persistence.
- Wrap internal errors with context when propagating through Go code.
- Keep protocol-compatibility logic near the public handlers instead of leaking management-style payloads into `/v1` routes.
- Validate config before persisting it. `PutConfigYAML` writes to a temp file, reloads it, and only then updates live state.

Primary evidence: `internal/api/handlers/management/config_basic.go`, `internal/store/*.go`, `sdk/api/handlers/handlers.go`

---

## API Error Responses

### Public compatibility routes

- Use protocol-compatible JSON payloads.
- Preserve upstream JSON when possible.
- Pick status codes that line up with the external client contract.

### Management routes

- Use standard HTTP error statuses.
- Prefer explicit operator messages over generic “bad request” text.
- Keep auth failures specific: missing key, invalid key, remote management disabled, banned IP, config unavailable.

### OAuth callbacks

OAuth callback endpoints persist code/state into the auth workspace and always return a success HTML page for the browser flow. Treat callback persistence failures as runtime/logging concerns, not user-facing JSON responses.

Primary evidence: `internal/api/server.go`, `internal/api/handlers/management/oauth_callback.go`

---

## Common Mistakes

- Do not return management-style payloads from OpenAI/Gemini/Claude/Codex-compatible routes.
- Do not leak raw secrets, tokens, or full upstream bodies in error messages.
- Do not skip validation before persisting config changes.
- Do not make path-guarded translator changes under the guise of “error handling cleanup”.
