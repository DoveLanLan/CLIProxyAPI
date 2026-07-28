# Tasks: bounded xAI retry through Resin lease rotation

- [x] Extend and normalize `XAIResinProxyConfig` with admin settings and bounded
      exact-402 retries.
- [x] Add a Resin admin client that resolves Platform IDs and deletes only the
      deterministic Account lease.
- [x] Add bounded exact-402 retry to non-stream execution and OAuth refresh.
- [x] Add safe replay handling for generic HTTP requests.
- [x] Add bootstrap-only exact-402 retry for SSE and handshake-only retry for
      WebSocket execution.
- [x] Merge the exact-402 and pre-response network retry paths with independent
      request budgets and combined-sequence regression coverage.
- [x] Preserve request-scoped failures, explicit-proxy precedence, no egress
      fallback, and secret redaction.
- [x] Update config diff output and deployment templates/scripts/docs.
- [x] Preserve an Actions-provided immutable image over server `.env` values and
      add a Docker/SSH-free shell regression for explicit and fallback cases.
- [x] Add config, admin API, non-stream, HTTP, SSE, WebSocket, refresh, limit,
      non-exact-402, and failure-path tests.
- [x] Run `gofmt -w .`.
- [x] Run focused package tests and relevant race tests.
- [x] Run `go test ./...`.
- [x] Run `go build -o test-output ./cmd/server && rm test-output`.
- [x] Confirm `internal/translator/**` is untouched and deployment assets contain
      no secret values.
- [x] Write summary, regression checklist, rollback notes, and quality gate.
