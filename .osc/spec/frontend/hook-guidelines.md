# Hook Guidelines

> What replaces web-framework hooks in this repository.

---

## Overview

There are no React/Vue hooks in this codebase. UI behavior is driven by:

- Bubble Tea messages and update loops in `internal/tui`
- plain Go methods on models and clients
- server-side configuration and Management API state

---

## Custom Hook Patterns

Not applicable as a primary pattern.

If you find yourself wanting a “hook”, choose one of these instead:

- a small helper function for pure transformation
- a method on the relevant TUI model
- a typed Bubble Tea message and command
- a shared client/helper package when the behavior is transport-related

---

## Data Fetching

- The TUI should fetch data through its client layer, not from scattered `http` calls inside every view path.
- Management data is authoritative on the backend; the TUI is only a client.
- External dashboards should consume the Management API from their own repositories.

---

## Naming Conventions

- Message types should describe an event or result.
- Commands should reflect an action boundary such as connect, refresh, save, or load.
- Avoid introducing frontend-framework vocabulary that does not match the Go/TUI model.

---

## Common Mistakes

- Do not document or implement fake “hooks” for a repo that does not use them.
- Do not put long-running network logic directly into rendering code.
- Do not mirror browser-framework state patterns when the backend already owns the source of truth.
