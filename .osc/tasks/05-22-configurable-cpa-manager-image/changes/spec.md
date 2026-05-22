# Spec: Make CPA-Manager Image Configurable

- Date: 2026-05-22
- Owner(s): hewei
- Related: proposal.md, tasks.md

## Repo Snapshot

- Modules/components: Go server under `cmd/` and `internal/`, SDK under `sdk/`, production deployment under `deploy/`.
- Toolchains: Go module `github.com/router-for-me/CLIProxyAPI/v7` with Go 1.26.0 in `go.mod`.
- Build/test/quality: PR build runs `go build -o test-output ./cmd/server`; AGENTS also requires `go test ./...` for full tests and `go build` after changes.
- CI: GitHub Actions in `.github/workflows/` build PRs and publish GHCR images for CLIProxyAPI.
- Runtime/deploy: `deploy/compose.production.yml` defines `cli-proxy-api` and external `cpa-manager` services on the `vps-gateway` network.
- Confidence: High.
- Evidence: `go.mod`, `.github/workflows/pr-test-build.yml`, `.github/workflows/docker-image.yml`, `deploy/compose.production.yml`, `deploy/README.md`, `AGENTS.md`.

## Scope

### In scope

- Add `CPA_MANAGER_IMAGE` as the production CPA-Manager image override.
- Document how to set a forked CPA-Manager image in `/opt/cliproxyapi/.env`.

### Out of scope

- CPA-Manager source, Dockerfile, or workflow changes.
- Changes to port mapping, volume paths, management key, upstream URL, or usage query limit.
- CLIProxyAPI application code changes.

## Acceptance Criteria (testable)

1. Production compose defaults to `seakee/cpa-manager:latest` when `CPA_MANAGER_IMAGE` is not set. (Verify: inspect `deploy/compose.production.yml`)
2. Operators can set a forked image through `.env` without editing compose. (Verify: variable uses `${CPA_MANAGER_IMAGE:-seakee/cpa-manager:latest}`)
3. Deployment docs list `CPA_MANAGER_IMAGE` and recommend fixed fork tags for production. (Verify: inspect `deploy/README.md`)
4. Existing `18318` CPA-Manager mapping and `/data` volume remain unchanged. (Verify: inspect `deploy/compose.production.yml`)

## Behavior / Requirements

The production compose file must read `CPA_MANAGER_IMAGE` only for the CPA-Manager container image. If the variable is unset, Docker Compose must render the same `seakee/cpa-manager:latest` image as before. The environment variables, private port binding, exposed port, volume, and network attachment must remain unchanged.

## Edge Cases

- Empty or invalid `CPA_MANAGER_IMAGE` values fail at Docker Compose pull/start time; no special handling is added.
- A private GHCR image requires a prior `docker login` on the VPS.
- Rollback should only require changing or removing `CPA_MANAGER_IMAGE` and restarting CPA-Manager.

## Compatibility Notes

- Backwards compatibility: Existing deployments with no `CPA_MANAGER_IMAGE` keep the current upstream image.
- Data/migrations: No data or SQLite schema changes.
- Config/flags: Adds one optional deployment environment variable.

## API/UX Decisions

- Inputs/outputs: `.env` may define `CPA_MANAGER_IMAGE=ghcr.io/<owner>/cpa-manager:sha-<commit>`.
- States/errors: Docker Compose handles image pull/start failures.
- Telemetry/logging: Not applicable.
- Accessibility/i18n: Not applicable.
