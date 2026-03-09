# Backend Development Guidelines

> Backend conventions for the Go proxy server, SDK runtime, and management surfaces in this repository.

---

## Overview

CLIProxyAPI is backend-heavy. Most repository changes land in one of these areas:

- HTTP protocol compatibility and route wiring
- auth loading, refresh, scheduling, and request execution
- config persistence and hot reload
- management APIs and operations tooling
- storage backends for config/auth state

Start with [Project Spec](../project-spec.md) for the high-level picture, then use the guides below for implementation rules.

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Package boundaries, where new code should go | Filled |
| [Database Guidelines](./database-guidelines.md) | State persistence and backing-store rules | Filled |
| [Error Handling](./error-handling.md) | Public API vs management API error shapes | Filled |
| [Quality Guidelines](./quality-guidelines.md) | Build/test/review expectations and forbidden changes | Filled |
| [Logging Guidelines](./logging-guidelines.md) | Global logging, request logs, and secret-handling rules | Filled |

---

## Scope Notes

- `cmd/server` is the production binary entrypoint.
- `internal/*` is server-private implementation.
- `sdk/*` is the embeddable surface shared with external integrators.
- `internal/tui` is not a web frontend; it is a terminal management client over the management API.

---

## Fast Reminders

1. Keep protocol-compatible behavior in the shared SDK/handler layers when it affects both the binary and embedders.
2. Keep management-only behavior under `internal/api/handlers/management`.
3. Preserve hot-reload, persistence, and version-metadata behavior when touching startup or config code.
4. Do not edit `internal/translator/**` in normal pull-request work.
