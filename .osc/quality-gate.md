# Quality Gate: Fix Codex OAuth Invalidated Token Failover

- Date: 2026-05-02

## Assumptions

- Change scope is backend auth metadata, auth-file synthesis, management Codex OAuth persistence, Codex refresh metadata, and auth manager failover.
- No `internal/translator/**` changes are expected or allowed.

## Suspected Change Scope

- `internal/auth/codex`
- `sdk/auth`
- `internal/watcher/synthesizer`
- `sdk/cliproxy/auth`
- `internal/runtime/executor`
- `internal/api/handlers/management`

## Detected Gates

- Gate Name: Go formatting. Confidence: High. Evidence: `AGENTS.md` requires `gofmt -w .` after Go changes.
- Gate Name: Focused Go tests. Confidence: High. Evidence: `.osc/spec/project-spec.md` expects focused tests for auth scheduling and protocol behavior changes.
- Gate Name: Server build. Confidence: High. Evidence: `AGENTS.md` and `.github/workflows/pr-test-build.yml` require `go build -o test-output ./cmd/server`.
- Gate Name: Translator path guard. Confidence: High. Evidence: `AGENTS.md` and `.github/workflows/pr-path-guard.yml` protect `internal/translator/**`.

## Suggested Gate Run (Local)

1. `gofmt -w <changed Go files>`: completed.
2. `go test ./sdk/cliproxy/auth`: passed.
3. `go test ./internal/watcher/synthesizer`: passed.
4. `go test ./sdk/auth`: passed.
5. `go test ./internal/auth/codex`: passed.
6. `go test ./internal/runtime/executor`: passed.
7. `go test ./internal/api/handlers/management`: passed.
8. `go build -o test-output ./cmd/server && rm test-output`: passed.

## Final Self-Review

- Security & secrets: no token logging added; new metadata fields are non-secret hashes or IDs already present in ID token claims.
- Edge cases & error handling: invalidated-token matching is narrow; generic `401` behavior remains temporary cooldown.
- Backward compatibility / migrations: existing auth files remain valid; new JSON fields are optional.
- API/contract compatibility: no public API route or config changes.
- Observability: persisted disabled reason is operator-readable.
- Config/env changes: none.
- Performance risk: low; JWT parsing was already used in these paths and remains request-independent except token refresh.
- Rollback plan: revert this task's changes; optional new auth JSON metadata can remain ignored.

## PR-ready checklist

- [x] `gofmt -w <changed Go files>`
- [x] `go test ./sdk/cliproxy/auth`
- [x] `go test ./internal/watcher/synthesizer`
- [x] `go test ./sdk/auth`
- [x] `go test ./internal/auth/codex`
- [x] `go test ./internal/runtime/executor`
- [x] `go test ./internal/api/handlers/management`
- [x] `go build -o test-output ./cmd/server && rm test-output`
- [x] Confirm no `internal/translator/**` files changed
