# Tasks: xAI dedicated rotating proxy pool

## Assumptions

- Production will initially configure six lane listeners and one probe listener.
- Subscription URLs are supplied later through the authenticated Management API
  and persist only in API-owned private files; the repository bootstrap contains
  no providers or live URLs.
- Local tests use fake controllers/proxies and do not contact xAI or public IP
  echo services.

## Checklist

- [x] 1. Add config schema and normalization.
  - Target: `internal/config/`, `config.example.yaml`
  - Change: opt-in xAI proxy-pool settings, lane/probe definitions, safe defaults,
    validation/sanitization, clone and parse coverage.
  - Verify: focused config tests.
  - Result: `go test ./internal/config` passed.

- [x] 2. Implement pool runtime and Mihomo controller client.
  - Target: `internal/runtime/executor/helps/`
  - Change: rendezvous routing, rollout, limiter, provider/node discovery,
    egress-IP probing/deduplication, rotation, quarantine state machines,
    atomic persistence, status and operator methods.
  - Verify: deterministic unit tests with fake clock/controller and fake HTTP API.
  - Result: pool/helper tests pass for routing, A/B promotion, quarantine,
    persistence, node failure, public-IP parsing, and queue overflow.

- [x] 3. Integrate all xAI execution paths.
  - Target: `internal/runtime/executor/xai*.go`, `sdk/api/handlers/`, and narrowly
    shared WebSocket session target helpers if required.
  - Change: per-request auth clones, no global proxy chaining, exact 402 A/B,
    pre-response retry only, midstream observation/no replay, refresh and generic
    `HttpRequest` routing, clean pool shutdown on executor replacement, and a
    CPA-managed `Retry-After` header for pool-local backpressure.
  - Verify: HTTP/SSE/WebSocket/refresh/executor tests and auth-manager neutral-error
    regression tests.
  - Result: executor tests pass for explicit proxy precedence, exact 402 retry,
    repeated credential 402, pre-payload stream retry, and no midstream replay;
    existing request-scoped auth-manager regressions remain in the focused suite.

- [x] 4. Route management inspection calls and add operator API.
  - Target: `internal/api/handlers/management/`, `internal/api/server.go`
  - Change: use registered xAI executor for xAI `api-call`; add status, provider
    refresh, lane rotate/check, and quarantine endpoints with redacted output.
  - Verify: handler and route-registration tests.
  - Result: management handler and server package tests pass; xAI `api-call`
    dispatch through the registered executor is covered.

- [x] 5. Add optional Mihomo deployment assets and documentation.
  - Target: `deploy/`, `deploy/scripts/remote-deploy.sh`, deployment docs
  - Change: unprivileged Compose overlay, secret-free multiple-provider example,
    persisted data paths, disabled-by-default env switch and operating notes.
  - Verify: `docker compose config` when Docker is available, shell syntax, secret
    scan, and manual port/capability review.
  - Result: shell syntax, the combined Compose render, Mihomo's native `-t`
    validation, secret scan, and manual capability/port review passed.

- [x] 6. Run closure and quality gates.
  - Target: task closure artifacts and `.osc/quality-gate.md`
  - Change: change summary, regression checklist, rollback notes, exact gate output.
  - Verify: gofmt, focused tests, `go test ./...`, required server build, translator
    path guard, git diff secret/self-review.
  - Result: closure artifacts were added; all required gates passed. Optional
    repository-wide vet/race findings are recorded as pre-existing baseline
    issues in `.osc/quality-gate.md`.

- [x] 7. Implement API-managed subscription registry and transactions.
  - Target: `internal/config/`, `internal/runtime/executor/helps/`, focused tests.
  - Change: optional registry/generated-config paths, versioned mode-`0600`
    storage, validation/redaction, fixed Mihomo YAML rendering, serialized
    candidate reload/verification/rollback, disable/drain/delete semantics.
  - Verify: store/render/controller transaction, rollback, corruption,
    concurrency, and secret-redaction tests including a pinned Mihomo container.
  - Result: transaction, fresh-provider verification, rollback, startup source
    recovery, delete/drain, redaction, limits, concurrency, and pinned-Mihomo
    tests pass.

- [x] 8. Add subscription Management API.
  - Target: `internal/api/handlers/management/`, `internal/api/server.go`, xAI
    executor operator surface.
  - Change: list/create/update/disable/check/delete endpoints, bounded bodies,
    write-only URLs, stable status codes, and redacted errors.
  - Verify: handler tests and route-registration coverage.
  - Result: CRUD/check handlers, strict `If-Match`, `ETag`, bounded bodies,
    disabled-state handling, and redacted error tests pass.

- [x] 9. Update optional deployment assets and operator docs.
  - Target: `deploy/compose.production.xai-proxy.yml`, Mihomo examples,
    `deploy/XAI_PROXY_POOL_SETUP_CN.md`, `config.example.yaml`.
  - Change: shared private config directory, API-owned source-of-truth workflow,
    bootstrap config, curl examples that avoid shell history, migration notes.
  - Verify: shell syntax, Compose render, Mihomo native config validation, file
    permission/path review, and secret scans.
  - Result: shell syntax, Compose rendering/security assertions, bootstrap and
    generated-config Mihomo validation, local hot reload, and secret scan pass.

- [x] 10. Re-run closure and quality gates for the API extension.
  - Target: task closure artifacts and `.osc/quality-gate.md`.
  - Change: update summary/regression/rollback notes and exact gate results.
  - Verify: gofmt, focused/race/integration tests, `go test ./...`, required
    server build, translator path guard, diff/security review.
  - Result: all required gates pass; repository-wide vet retains only six
    documented pre-existing findings.

## Notes

- Update this checklist as each independently verifiable step completes.
- Scope changes must update proposal/spec before further source edits.
