# Tasks: Fix Codex OAuth Invalidated Token Failover

- Date: 2026-05-02
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`

## Assumptions

- Desired runtime behavior is isolate-and-switch for exact Codex OAuth invalidated-token responses.
- The shared team `chatgpt_account_id` must remain valid and must not be used alone for uniqueness.

## Checklist

- [x] 1) Locate Codex auth metadata and manager failover paths
  - Target: `sdk/auth`, `sdk/cliproxy`, `internal/runtime/executor`
  - Change: identify exact symbols for auth file synthesis, disabled state persistence, and credential retry classification
  - Verify: code search and focused file reads

- [x] 2) Enrich Codex auth identity metadata
  - Target: Codex OAuth auth file synthesis/load path
  - Change: parse safe `id_token` identity fields and persist optional account/user/hash/plan metadata while preserving `account_id`
  - Verify: focused auth metadata tests

- [x] 3) Add invalidated-token quarantine behavior
  - Target: auth manager credential failure path
  - Change: classify exact invalidated-token `401`, persist disabled state/reason, and force same-request failover without reusing the auth
  - Verify: manager failover tests, including `max-retry-credentials=1`

- [x] 4) Preserve generic 401 cooldown behavior
  - Target: auth manager credential failure path
  - Change: keep generic `401` on recoverable cooldown path
  - Verify: regression test

- [x] 5) Format and verify
  - Target: changed Go files and repo gates
  - Change: run `gofmt -w` on changed Go files, focused tests, and required server build
  - Verify: `go test` focused packages; `go build -o test-output ./cmd/server && rm test-output`

## Notes

- Focused tests passed for `./sdk/cliproxy/auth`, `./internal/watcher/synthesizer`, `./sdk/auth`, `./internal/auth/codex`, `./internal/runtime/executor`, and `./internal/api/handlers/management`.
- Required compile gate passed with `go build -o test-output ./cmd/server && rm test-output`.
