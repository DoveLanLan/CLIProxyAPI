# Regression Checklist: Fix Codex OAuth Invalidated Token Failover

- Date: 2026-05-02
- Related: `spec.md`, `tasks.md`

## Gates (from Repo Snapshot)

- Build: `go build -o test-output ./cmd/server && rm test-output`
- Tests: focused `go test` packages for auth manager, auth synthesis, auth store, Codex auth helpers, runtime executor, and management handlers
- Lint/format: `gofmt -w` on changed Go files
- Other: verify no `internal/translator/**` files changed

## Manual checks (if applicable)

- Codex same-team login files: two users with the same `chatgpt_account_id` remain separate active auths.
- Invalidated token runtime path: first Codex auth with exact invalidated-token `401` is disabled and a good Codex auth handles the request.
- Generic `401`: auth is not persistently disabled and uses temporary cooldown state.

## Edge-case re-tests

- Existing Codex auth files without the new metadata fields still synthesize from `id_token` or remain valid without it.
- `max-retry-credentials=1` does not stop failover after quarantining an invalidated Codex token.
