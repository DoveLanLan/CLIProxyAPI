# Quality Guidelines

> Code quality standards for backend work in CLIProxyAPI.

---

## Overview

Backend quality in this repo is driven by a mix of CI gates, runtime compatibility constraints, and repository-specific boundaries. The build gate is lightweight, so review discipline matters.

---

## Forbidden Patterns

- Editing `internal/translator/**` in a normal pull request.
- Returning management-style payloads from public compatibility routes.
- Bypassing config validation or comment-preserving persistence in management config writes.
- Adding server-only behavior into `sdk/*` without confirming it belongs in the embeddable surface.
- Moving reusable protocol logic into `internal/*` where embedders can no longer share it.

---

## Required Patterns

- Keep `cmd/server` buildable with Go `1.26`.
- Preserve linker-injected version metadata used by release automation.
- Respect hot-reload/watch flows for config and auth changes.
- Keep management auth restrictions intact: localhost shortcut only when configured, remote access only with explicit secret and policy.
- Use the existing route-module pattern for optional route bundles when it fits.

---

## Testing Requirements

- Minimum required gate: `go build -o test-output ./cmd/server`
- For behavior changes, run affected package tests and relevant top-level regressions under `test/`.
- If a change touches auth scheduling, selection, or persistence, check the extensive tests under `sdk/cliproxy/auth/`.
- If a change touches request logging or middleware, check `internal/logging/*_test.go` and `internal/api/middleware/*_test.go`.

Even though CI currently shows only a build job for pull requests, treat missing focused tests as a real review gap.

---

## Code Review Checklist

1. Does the change preserve OpenAI/Gemini/Claude/Codex compatibility where required?
2. Is the boundary between `internal/*` and `sdk/*` still coherent?
3. Are config writes validated, persisted safely, and still reloadable?
4. Could logs or error messages leak secrets?
5. Are watcher and hot-reload flows still correct after persistence or auth changes?
6. Are release/build scripts still valid if startup or metadata behavior changed?
