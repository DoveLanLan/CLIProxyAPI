# Quality Gate: Integrate CPA-Manager Panel and Usage Monitoring

## Assumptions

- The change scope is backend management APIs, usage accounting, management-panel asset configuration, and Docker deployment examples.
- No `internal/translator/**` source files are intentionally changed.

## Suspected Change Scope

- Backend/runtime: `internal/redisqueue`, `internal/api/handlers/management`, `internal/api/server.go`, `sdk/cliproxy/*`, `internal/runtime/executor/helps`.
- Config/deploy: `internal/config`, `config.example.yaml`, compose files, `deploy/README.md`.
- UI integration: management panel asset source only; no in-repo browser app.

## Detected Gates

- Gate Name: Go build
  - Confidence: High
  - Evidence: `.github/workflows/pr-test-build.yml` runs `go build -o test-output ./cmd/server`.
- Gate Name: Go tests
  - Confidence: Medium
  - Evidence: `AGENTS.md` lists `go test ./...`; many packages include `*_test.go`.
- Gate Name: Go format
  - Confidence: High
  - Evidence: `AGENTS.md` requires `gofmt -w .` after Go changes.
- Gate Name: Translator path guard
  - Confidence: High
  - Evidence: `.github/workflows/pr-path-guard.yml` rejects `internal/translator/**` changes.

## Suggested Gate Run (Local)

1. `gofmt -w <changed-go-files>` - required after Go edits.
2. `go test ./...` - validates package and cross-module regressions.
3. `go build -o test-output ./cmd/server && rm test-output` - mirrors PR build gate.
4. `git diff --name-only -- internal/translator` - confirm protected path is untouched.

## Actual Gate Results

- `gofmt -w ...`: passed.
- `go test ./internal/redisqueue ./internal/api/handlers/management ./sdk/cliproxy/auth ./sdk/cliproxy/usage ./internal/usage`: passed.
- `go test ./...`: passed.
- `go build -o test-output ./cmd/server && rm test-output`: passed.
- `git status --short` shows no `internal/translator/**` changes.

## Final Self-Review

- Security & secrets: no secrets committed; `CPA_MANAGEMENT_KEY` is documented as operator-provided env/config.
- Edge cases & error handling: invalid usage queue count returns 400 without popping; disabled management clears queue.
- Backward compatibility / migrations: no schema migration; panel repository override remains supported.
- API/contract compatibility: new management endpoints are under existing auth middleware.
- Observability: config diff reports retention changes; queue data is available through management API.
- Config/env changes: `redis-usage-queue-retention-seconds`, `CPA_MANAGER_CPA_URL`, `CPA_MANAGEMENT_KEY`, and `TAILSCALE_CPA_MANAGER_PORT` are documented.
- Performance risk: queue is in-memory, bounded by retention time, and only active when management is enabled.
- Rollback plan: revert this change and stop/remove the external CPA-Manager service.

## PR-ready checklist

- [x] `gofmt -w <changed-go-files>`
- [x] `go test ./...`
- [x] `go build -o test-output ./cmd/server && rm test-output`
- [x] Protected translator path check: no `internal/translator/**` changes.
