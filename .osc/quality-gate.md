# Quality gate: standalone EgressProxyPool extraction

Date: 2026-07-27
Task: `.osc/tasks/07-27-extract-egress-proxy-pool`
Status: PASS with one pre-existing unrelated race-test finding

## Changed scope

- New sibling Git project: `/root/Projects/Go/src/EgressProxyPool`.
- Standalone pool/controller/subscription runtime and authenticated `/v1` API.
- CLIProxyAPI remote pool client and simplified configuration.
- CLIProxyAPI Management API compatibility facade and deployment integration.
- No changes under `internal/translator/**`.

## Required and focused gates

| Project / command | Result |
|---|---|
| EgressProxyPool `gofmt -w .` | PASS |
| EgressProxyPool `go test ./...` | PASS |
| EgressProxyPool `go test -race ./internal/api ./internal/pool` | PASS |
| EgressProxyPool `go vet ./...` | PASS |
| EgressProxyPool `go build -o test-output ./cmd/server && rm test-output` | PASS |
| EgressProxyPool Docker image build | PASS |
| CLIProxyAPI `gofmt -w .` | PASS |
| CLIProxyAPI `go test ./...` | PASS |
| CLIProxyAPI focused xAI/helper/management race tests | PASS |
| CLIProxyAPI `go vet` on changed packages | PASS |
| CLIProxyAPI `go build -o test-output ./cmd/server && rm test-output` | PASS |
| `git diff --check` and `bash -n deploy/scripts/remote-deploy.sh` | PASS |

The Docker build needed the host HTTPS proxy for dependency download after one
direct `proxy.golang.org` timeout; the resulting image compiled successfully.

## Deployment gates

| Check | Result |
|---|---|
| EgressProxyPool `docker compose config` | PASS |
| CLIProxyAPI base + standalone-pool overlay Compose render | PASS |
| No controller/Mihomo host port publication | PASS |
| No host network, privileged mode, TUN, or `cap_add` | PASS |
| Both containers use `cap_drop: ALL` and `no-new-privileges` | PASS |
| CLIProxyAPI mounts only the standalone API token | PASS |

The expected existing CLIProxyAPI Tailscale management host port remains in the
combined render; the pool overlay adds no published port.

## Security and compatibility review

- PASS: constant-time bearer-token comparison and bounded strict JSON bodies.
- PASS: raw auth IDs are never sent; route keys are HMAC-SHA256 digests.
- PASS: client transport bypasses environment/global proxies for private control
  traffic and never logs tokens or request bodies.
- PASS: subscription URLs remain write-only and errors are redacted.
- PASS: registry/generated files retain atomic mode-`0600` behavior.
- PASS: probe leases are random, single-use, expiring, and released on shutdown.
- PASS: established xAI streams get no new read deadline and are not replayed
  after downstream payload.
- PASS: explicit auth proxy precedence and request-scoped managed errors remain.
- PASS: protected `internal/translator/**` is unchanged.
- PASS: no production host, configuration, subscription, or service was changed.

## Pre-existing race finding

Running race mode for the entire `internal/runtime/executor` package reports an
existing Antigravity credits test-cleanup race between
`resetAntigravityCreditsRetryState` and background credit-refresh goroutines in
`antigravity_executor_credits_test.go` / `antigravity_executor.go`. Those files
are outside this change. The xAI proxy-pool executor tests, helper client tests,
and management tests pass under race mode.
