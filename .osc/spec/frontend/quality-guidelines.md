# Quality Guidelines

> UI quality standards for the TUI and management-panel integration code.

---

## Overview

UI work in this repository is operational UX work. The main quality bar is correctness, keyboard usability, and consistency with the Management API contract.

---

## Forbidden Patterns

- Pretending this repo contains a normal web frontend when it does not
- Embedding a hand-maintained browser dashboard into the server without an explicit architecture change
- Bypassing the TUI client and scattering raw management HTTP calls throughout tab code
- Hardcoding duplicated style palettes across tabs

---

## Required Patterns

- Keep TUI flows keyboard-accessible and status-readable.
- Keep UI code aligned with Management API behavior and auth rules.
- When changing the management panel integration, preserve updater fallback and static-path behavior.
- When changing tabs, preserve lazy initialization and screen resizing behavior in the root app.

---

## Testing Requirements

- For TUI client or management integration changes, prefer focused package tests where feasible and at minimum verify the server-side management contract the UI depends on.
- For management-panel asset changes, verify the updater logic still supports download, fallback, and path resolution.
- UI changes must not break the repository’s minimum server build gate.

---

## Code Review Checklist

1. Does the UI change rely on real server/management behavior instead of a local-only assumption?
2. Are styles and status messages consistent with the existing TUI?
3. Could the change expose secrets in the terminal or management asset path?
4. If the change touches management endpoints, was the backend contract reviewed at the same time?
