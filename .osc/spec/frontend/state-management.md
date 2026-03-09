# State Management

> How UI state is managed in this repository.

---

## Overview

There is no browser-side global state container here. State falls into two buckets:

- transient TUI state in Bubble Tea model structs
- persistent backend state in config/auth storage exposed via the Management API

---

## State Categories

### Local UI state

Examples:

- active tab
- terminal dimensions
- auth gate text input
- initialized-tab flags
- logs view toggles

Primary evidence: `internal/tui/app.go`

### Persistent backend state

Examples:

- config values from `config.yaml`
- auth files under the auth directory or mirrored backing stores
- usage statistics and request-log settings

Primary evidence: `config.example.yaml`, `internal/api/handlers/management/`, `internal/store/`

---

## When to Use Shared UI State

Use shared TUI state only when multiple tabs need coordinated awareness, such as:

- login/authenticated state
- whether log-related tabs should exist
- common terminal width/height information

Keep it on the root app model unless there is a strong reason to centralize elsewhere.

---

## Server State

- The backend is the source of truth.
- The TUI should refresh from the Management API rather than inventing local persistence.
- External control panels should use the same principle.

---

## Common Mistakes

- Do not duplicate backend config as an independent client-side state system.
- Do not store durable UI data in a way the Management API cannot see.
- Do not let stale TUI state outlive a backend config change without an explicit refresh path.
