# hewei journal 7

- Date: 2026-05-02
- Title: Fix Codex OAuth invalidated token failover
- Commit:

## Summary

Conclusions/decisions: `chatgpt_account_id` is a ChatGPT account/workspace identifier and can be shared by multiple team users, so it remains as `account_id` only for `Chatgpt-Account-Id` header compatibility. New optional non-secret Codex metadata records the account/user split and stable short hashes.

What changed: added `internal/auth/codex/identity.go`, wired Codex identity metadata into login, management OAuth save, file synthesis/load, and token refresh, and added exact Codex `401` + `invalidated oauth token` handling that disables only the failing auth with `disabled_reason: codex oauth token invalidated; re-login required`.

Verification: ran `gofmt`; `go test ./sdk/cliproxy/auth`; `go test ./internal/watcher/synthesizer`; `go test ./sdk/auth`; `go test ./internal/auth/codex`; `go test ./internal/runtime/executor`; `go test ./internal/api/handlers/management`; and `go build -o test-output ./cmd/server && rm test-output`. All passed.

Risks/rollback: optional new auth JSON metadata should be backward-compatible, but persisted invalidated-token quarantines leave `disabled: true` until re-login or manual enable. Rollback is a straight revert of this task's code/tests/docs; invalidated tokens still require re-login.
