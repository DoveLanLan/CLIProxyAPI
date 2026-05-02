# Spec: Fix Codex OAuth Invalidated Token Failover

- Date: 2026-05-02
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## Repo Snapshot (from step 0)

- Modules/components: `cmd/server` runtime entrypoint, `internal/*` server-only code, `sdk/*` reusable embedding/auth/protocol surface, `test/` cross-package regressions. Confidence: High. Evidence: `cmd/server/main.go`, `internal/`, `sdk/`, `test/`.
- Toolchains: Go modules with `go test ./...` package tests and required `go build -o test-output ./cmd/server` compile gate. Confidence: High. Evidence: `go.mod`, `config.example.yaml`, `.github/workflows/pr-test-build.yml`, `AGENTS.md`.
- Quality/CI: PR build gate compiles `cmd/server`; `internal/translator/**` is protected and out of scope. Confidence: High. Evidence: `.github/workflows/pr-test-build.yml`, `.github/workflows/pr-path-guard.yml`, `AGENTS.md`.

## Scope

### In scope

- Codex OAuth auth metadata extraction and file synthesis.
- Auth manager classification and persistence for exact invalidated-token errors.
- Credential rotation/failover behavior for Codex OAuth invalidation.
- Focused unit/regression coverage.

### Out of scope

- `internal/translator/**`.
- Public API routes or config schema changes.
- Proxy URL parsing or split-proxy behavior.
- Deleting auth files by shared team hash prefix.

## Acceptance Criteria (testable)

1. Two Codex team auth files with the same `chatgpt_account_id` but different `chatgpt_user_id` or email synthesize as distinct active auth files. Verify with focused auth tests.
2. A Codex `401` error containing `invalidated oauth token` disables only the failing auth, persists `disabled: true` with a non-secret disabled reason, and routes the same request to another Codex auth. Verify with focused manager tests.
3. A generic Codex `401` keeps the existing temporary cooldown behavior and does not persistently disable the auth. Verify with regression test.
4. `max-retry-credentials: 1` still falls through after invalidated-token quarantine and never retries the same invalidated credential during the request. Verify with focused manager test.
5. Existing Codex auth files without new metadata fields still load. Verify via existing tests and focused package tests.
6. Server build passes with `go build -o test-output ./cmd/server && rm test-output`.

## Behavior / Requirements

- `account_id` remains the value used for `Chatgpt-Account-Id` compatibility.
- New non-secret metadata fields may be written: `chatgpt_account_id`, `chatgpt_user_id`, `codex_account_hash`, `codex_user_hash`, `plan_type`, and `disabled_reason`.
- Invalidated-token classification requires HTTP status `401` and a case-insensitive message containing `invalidated oauth token`.
- Quarantined auths are disabled immediately, persisted, and excluded from further attempts in the same request.
- Generic `401` responses are treated as recoverable credential failures using the existing cooldown path.

## Edge Cases

- Missing or malformed `id_token`: keep existing compatibility behavior and avoid failing auth synthesis solely for optional metadata.
- Multiple users in one ChatGPT team: file identity must include user-specific material so they do not overwrite or dedupe each other.
- Persistence failure while disabling: log a non-secret warning/error but do not expose token material.
- All credentials invalidated or unavailable: return the normal final upstream failure once failover options are exhausted.

## Compatibility Notes

- Backwards compatibility: Existing auth files are valid; added fields are optional.
- Data/migrations: No migration is required.
- Config/flags: No new config keys or flags.

## API/UX Decisions (if applicable)

- Inputs/outputs: No public API route or request/response schema changes.
- States/errors: Disabled auths should expose an operator-readable status such as `codex oauth token invalidated; re-login required`.
- Telemetry/logging: Log credential identifier/status only; never log tokens or full JWTs.
