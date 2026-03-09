# Directory Structure

> How the UI-related code is organized in this repository.

---

## Overview

There is no `web/`, `app/`, or `frontend/` browser application here. UI-related code is split between a terminal UI and an externally sourced management panel asset.

---

## Directory Layout

```text
internal/
  tui/
    app.go                root Bubble Tea model and tab switching
    *_tab.go              per-tab UI state and update/view logic
    styles.go             shared Lip Gloss styles
    client.go             Management API client for the TUI
    browser.go            browser-launch helpers for auth flows

  managementasset/
    updater.go            download/check/serve management.html from external release source
```

---

## Module Organization

### Use `internal/tui` when

- the feature is keyboard-driven local management UX
- it talks to the Management API over HTTP
- it belongs in the bundled terminal application

### Use `internal/managementasset` when

- the change affects where `management.html` is downloaded from
- the update cadence, fallback logic, or static asset path changes
- the server should serve or refresh the external control panel differently

### Do not add an in-repo web app when

- the goal is simply to expose more management functionality

In that case, extend the Management API and let external dashboards consume it.

---

## Naming Conventions

- Keep TUI files focused by screen or tab (`config_tab.go`, `logs_tab.go`, `usage_tab.go`).
- Keep shared TUI styles centralized in `styles.go`.
- Keep management-panel asset logic under `internal/managementasset` instead of scattering download logic across handlers.

---

## Examples

- Root TUI model and tab routing: `internal/tui/app.go`
- Shared terminal styles: `internal/tui/styles.go`
- External management asset syncing: `internal/managementasset/updater.go`
