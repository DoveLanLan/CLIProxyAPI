# Frontend Development Guidelines

> UI conventions for the terminal UI and management-panel integration in this repository.

---

## Overview

This repository does not contain a first-party React/Vue/Next.js frontend application. The in-repo UI surfaces are:

- the Bubble Tea terminal UI under `internal/tui`
- the management panel asset updater/serving logic under `internal/managementasset`

External dashboards mentioned in the README live in other repositories and integrate through the Management API.

Use this directory only when you are changing:

- terminal UI tabs, styles, flows, or management client behavior
- how the server fetches, stores, or serves `management.html`
- contracts that external dashboards depend on through the Management API

---

## Guidelines Index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | TUI and management-panel integration layout | Filled |
| [Component Guidelines](./component-guidelines.md) | Bubble Tea tab/model conventions | Filled |
| [Hook Guidelines](./hook-guidelines.md) | What replaces web hooks in this repo | Filled |
| [State Management](./state-management.md) | TUI state vs backend state | Filled |
| [Quality Guidelines](./quality-guidelines.md) | UI-specific review expectations | Filled |
| [Type Safety](./type-safety.md) | Go-side typing and validation for UI code | Filled |

---

## Scope Notes

- If you need a new web dashboard, it probably belongs in a separate repository.
- If you need a local operator UX in this repo, prefer extending the TUI or Management API.
- Do not hand-maintain a bundled browser app under this repo without an explicit product decision.
