# Quality Gate: Fix CPA-Manager Monitoring Load

## Assumptions

- The code change is deployment/documentation only.
- No Go source files, generated assets, `.github` workflows, Docker build files, or `internal/translator/**` paths are intentionally touched.
- The VPS deploy path uses `deploy/compose.production.yml` or a compatible compose environment.

## Suspected Change Scope

- Deployment: `deploy/compose.production.yml`
- Documentation: `deploy/README.md`
- Workflow artifacts: `.osc/tasks/05-22-fix-cpa-manager-monitoring-load/changes/*`

## Detected Gates

- Gate Name: Compose config render
  - Confidence: High
  - Evidence: `deploy/compose.production.yml` defines the affected service and environment variables.
- Gate Name: Deployment documentation review
  - Confidence: High
  - Evidence: `deploy/README.md` documents the VPS environment variables and CPA-Manager access notes.
- Gate Name: Go build
  - Confidence: Low for this change
  - Evidence: `AGENTS.md` requires `go build -o test-output ./cmd/server && rm test-output` after code changes; no Go files are modified here.
- Gate Name: Protected path review
  - Confidence: High
  - Evidence: `AGENTS.md` and project spec protect `internal/translator/**`; user also requested not to touch `.github` or Docker build-related files in the upstream sync context.

## Suggested Gate Run (Local)

1. `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test docker compose -f deploy/compose.production.yml config` - verify compose renders and `USAGE_QUERY_LIMIT` expands.
2. `git diff --name-only` - verify only intended deployment/docs/workflow paths changed.
3. `git diff --check` - verify no whitespace or patch formatting issues.
4. `go build -o test-output ./cmd/server && rm test-output` - optional for this deployment-only task, required before merging if repository policy treats any non-doc change as needing a server build.

## Actual Gate Results

- PASS: `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test docker compose -f deploy/compose.production.yml config` rendered successfully and included `USAGE_QUERY_LIMIT: "100"`.
- PASS: `git diff --name-only` showed only `.osc/quality-gate.md`, `deploy/README.md`, and `deploy/compose.production.yml` among tracked changes.
- PASS: `git diff --check`.
- PASS: `go build -o test-output ./cmd/server && rm test-output`.

## Final Self-Review

- Security & secrets: no management key, Authorization header, auth file, or local `config.yaml` value was committed.
- Edge cases & error handling: docs cover lowering the limit further if recent rows still make `/v0/management/usage` time out.
- Backward compatibility / migrations: existing behavior can be restored with `CPA_MANAGER_USAGE_QUERY_LIMIT=50000`; no data migration.
- API/contract compatibility: no CLIProxyAPI public API or SDK contract changed.
- Observability: no logging changes.
- Config/env changes: new `CPA_MANAGER_USAGE_QUERY_LIMIT` deployment variable is documented.
- Performance risk: default query window is reduced aggressively to improve page-load latency at the cost of exhaustive historical display.
- Rollback plan: remove `USAGE_QUERY_LIMIT` from compose or set `CPA_MANAGER_USAGE_QUERY_LIMIT=50000`, then restart `cpa-manager`.

## PR-ready checklist

- [x] `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test docker compose -f deploy/compose.production.yml config`
- [x] `git diff --name-only`
- [x] `git diff --check`
- [x] Optional: `go build -o test-output ./cmd/server && rm test-output`
