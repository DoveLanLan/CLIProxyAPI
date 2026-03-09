# Component Guidelines

> How UI components are built in this repository.

---

## Overview

“Components” here means terminal UI models and views, not React components.

The dominant pattern is Bubble Tea:

- a root app model owns shared navigation and auth gating
- each tab owns its own update/view behavior
- shared visual language lives in Lip Gloss styles

Primary evidence: `internal/tui/app.go`, `internal/tui/styles.go`

---

## Component Structure

For new TUI features:

- add a focused model for the screen/tab
- keep `Init`, `Update`, and `View` responsibilities clear
- let the root app coordinate tab switching and lazy initialization
- keep transport logic behind the TUI client rather than embedding raw HTTP everywhere

---

## Props Conventions

There are no JSX props here. The equivalent rules are:

- pass explicit dependencies into models at construction time
- keep shared client/config references on the model struct
- prefer typed messages over loosely shaped shared state

---

## Styling Patterns

- Reuse shared Lip Gloss styles from `internal/tui/styles.go`.
- Keep palette and status-color choices centralized.
- Avoid each tab inventing its own ad hoc color set unless there is a clear UX reason.

---

## Accessibility

- Keep the TUI keyboard-first.
- Make auth errors and status changes visible in plain text, not color alone.
- Preserve readable widths and clear labels for narrow terminal sizes.

---

## Common Mistakes

- Do not treat the downloaded `management.html` asset as a hand-edited local component.
- Do not duplicate style constants in each tab.
- Do not let one tab mutate unrelated tab state directly when the root app should coordinate it.
