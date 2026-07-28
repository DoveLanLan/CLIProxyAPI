# Quality gate: native xAI Resin routing

Date: 2026-07-28
Task: `.osc/tasks/07-27-xai-resin-routing`
Status: PASS with one pre-existing unrelated race-test finding

## Changed scope

- New optional `xai-resin-proxy` config and SDK config alias.
- Runtime HMAC identity and credentialed forward-proxy URL helper.
- xAI HTTP, SSE, WebSocket, Management HTTP, and refresh routing integration.
- Request-scoped Resin configuration and transport errors.
- Redacted config change reporting, production overlay, deploy validation, and
  Chinese two-project setup documentation.
- No changes under `internal/translator/**`.

## Required and focused gates

| Command | Result |
|---|---|
| `gofmt -w .` | PASS |
| `go test ./internal/config ./internal/runtime/executor/helps ./internal/runtime/executor ./internal/watcher/diff -count=1` | PASS |
| `go test ./...` | PASS |
| `go test -race -run 'Resin' ./internal/runtime/executor -count=1` | PASS |
| `go test -race -run 'XAIResin' ./internal/runtime/executor/helps -count=1` | PASS |
| `go vet ./internal/config ./internal/runtime/executor/helps ./internal/runtime/executor ./internal/watcher/diff ./sdk/config` | PASS |
| `go build -o test-output ./cmd/server && rm test-output` | PASS |
| `bash -n deploy/scripts/remote-deploy.sh` | PASS |
| Base + Resin overlay `docker compose ... config` | PASS |
| `git diff --check` | PASS |
| `git diff --name-only -- internal/translator` is empty | PASS |

## Security and compatibility review

- PASS: one config block covers large xAI credential sets without auth-file edits.
- PASS: raw auth IDs and xAI credentials are not sent to Resin.
- PASS: the proxy token and identity key are file-backed and absent from config diffs.
- PASS: derived proxy URLs are in-memory only and removed after refresh.
- PASS: explicit auth proxies preserve their existing highest priority.
- PASS: enabled Resin never falls back to EgressProxyPool or CPA global routing.
- PASS: Resin infrastructure failures do not cool otherwise valid xAI auths.
- PASS: HTTP proxy auth and per-auth WebSocket target isolation are covered.
- PASS: non-xAI providers and translator implementations are unchanged.
- PASS: production CPA and Resin containers are running on `vps-gateway`; CPA
  returns HTTP 200 and Resin reports healthy.
- PASS: the production secret files are mode `0600` and mounted into CPA read-only.
- PASS: Resin is enabled, EgressProxyPool is disabled, and no auth file contains
  a generated Resin proxy URL.
- PASS: real CPA requests produced 89 valid anonymous xAI leases in Resin, with
  no malformed xAI lease names.
- PASS: sampled traffic reached xAI through Resin. xAI returned
  `402 personal-team-blocked:spending-limit`, which is an upstream credential
  state rather than an infrastructure failure.
- PASS: the pre-rollout production backup is present at
  `/opt/cliproxyapi/backups/xai-resin-20260727T223545Z`.
- PASS: an exhaustive, non-mutating check covered all 1,291 enabled xAI
  credentials and their independently derived Resin Accounts.
- PASS: all 1,291 expected Accounts were observed in Resin, with zero missing
  and zero malformed xAI Account names.
- PASS: CPA remained HTTP 200 and Resin remained healthy after the exhaustive
  check.
- UPSTREAM CREDENTIAL RESULT: no enabled credential completed `grok-3-mini`
  with HTTP 200. The final unique-credential results were 1,283 xAI
  spending-limit 402 responses and 8 current-token 401 responses.
- PASS: after operator-confirmed evacuation, all 4,828 xAI credential files are
  outside the active auth directory; both source and runtime xAI counts are
  zero while all 17 unrelated auth files remain active.
- PASS: the exact post-migration archive contains 4,828 files and passes its
  SHA-256 check; CPA is HTTP 200 and the five-minute inspection timer remains
  active.
- PASS: a manual inspection run with zero runtime xAI auths completed
  successfully and applied no credential action.

## Pre-existing race finding

Running `go test -race ./internal/runtime/executor ./internal/runtime/executor/helps`
reports an existing Antigravity credits test-cleanup race between
`resetAntigravityCreditsRetryState` and background credit-refresh goroutines in
`antigravity_executor_credits_test.go` / `antigravity_executor.go`. Those files
are outside this task. Both Resin-focused executor race suites and the complete
helper package result for this change pass.
