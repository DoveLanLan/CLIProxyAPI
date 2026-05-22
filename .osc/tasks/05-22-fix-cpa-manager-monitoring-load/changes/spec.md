# Spec: Fix CPA-Manager Monitoring Load

- Date: 2026-05-22
- Owner(s): hewei
- Related: proposal.md, tasks.md

## Repo Snapshot (from step 0)

- Modules/components: `cmd/server` server entrypoint, `internal/*` private runtime packages, `sdk/*` reusable embedding/API surface, `deploy/*` VPS deployment artifacts, `.osc/*` workflow artifacts.
- Toolchains: Go modules with `go test ./...` and `go build -o test-output ./cmd/server`; deployment via Docker Compose files under `deploy/`.
- Style/format: Go changes require `gofmt`; task changes require `.osc/tasks/<task>/changes/` artifacts; avoid protected `internal/translator/**`.
- CI gates: build `./cmd/server`; path guard protects `internal/translator/**`; release workflows build Docker/Goreleaser artifacts.
- Confidence: High, based on `AGENTS.md`, `.osc/spec/project-spec.md`, `go.mod`, `deploy/README.md`, and `.github/workflows/*`.

## Scope

### In scope

- Add a configurable `USAGE_QUERY_LIMIT` environment variable to the production `cpa-manager` service.
- Document the default and the observed browser/backend behavior in deployment notes.
- Verify the changed YAML remains syntactically valid enough for review and the repo remains clean aside from intended files.

### Out of scope

- CPA-Manager source changes.
- Docker image build changes.
- Data deletion, migration, or automated SQLite maintenance on the VPS.
- CLIProxyAPI runtime or API behavior changes.

## Acceptance Criteria (testable)

1. Production compose passes an explicit `USAGE_QUERY_LIMIT` to the `cpa-manager` container. (Verify: inspect `deploy/compose.production.yml`)
2. Operators can override the limit from `.env` without editing compose. (Verify: variable uses `${CPA_MANAGER_USAGE_QUERY_LIMIT:-100}`)
3. Deployment docs explain why the limit exists and how to lower it if the panel still times out. (Verify: inspect `deploy/README.md`)
4. No `.github`, Docker build file, or `internal/translator/**` path is modified. (Verify: `git diff --name-only`)

## Behavior / Requirements

CPA-Manager should query a bounded recent event window for the monitoring dashboard. The default limit should favor fast page loads over exhaustive historical totals because the dashboard currently blocks on `/v0/management/usage` and the external service does not support server-side pagination through the frontend.

## Edge Cases

- If the latest events contain unusually large raw payloads, operators may need to lower the limit further.
- If historical totals are required, operators can temporarily raise the limit or use upstream tooling once CPA-Manager fixes pagination/export behavior.
- Existing SQLite data is left intact; this change only affects future dashboard queries after the container restarts.

## Compatibility Notes

- Backwards compatibility: Existing deployments can keep their current behavior by setting `CPA_MANAGER_USAGE_QUERY_LIMIT=50000`.
- Data/migrations: No schema or data migration.
- Config/flags: Adds one deployment environment variable only.

## API/UX Decisions (if applicable)

- Inputs/outputs: No API contract changes in CLIProxyAPI.
- States/errors: CPA-Manager upstream should eventually settle loading state when usage requests time out under auto-refresh.
- Telemetry/logging: Do not log management keys or captured Authorization headers.
- Accessibility/i18n: No in-repo UI text changes.
