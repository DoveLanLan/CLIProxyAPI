# Spec: Docker Compose Rebuild Fix

- Date: 2026-06-08
- Owner(s): Codex
- Related: proposal.md, tasks.md

## Repo Snapshot (from step 0)
- Modules/components: Go server under `cmd/` and `internal/`, Docker runtime via `Dockerfile` and `docker-compose.yml`.
- Toolchains: Docker Compose build, Go module download/build.
- Confidence: High.
- Evidence locations: `Dockerfile`, `docker-compose.yml`, `go.mod`, `go.sum`, `AGENTS.md`.

## Scope
### In scope
- Add the missing builder-stage package needed by Go module resolution.
- Re-run Docker Compose build and recreate the service container.

### Out of scope
- Runtime image package changes.
- Application source changes.
- Management UI behavior analysis.

## Acceptance Criteria (testable)
1. `docker compose build cli-proxy-api` passes `go mod download`. (Verify: command exits 0.)
2. The `cli-proxy-api` container is recreated from the newly built image. (Verify: `docker compose up -d --force-recreate cli-proxy-api` exits 0 and container is running.)

## Behavior / Requirements
The builder image must include the tools required for Go module downloads. The final runtime image should continue to include only runtime dependencies already present before this change.

## Edge Cases
- If a future Go dependency also needs SSH or private credentials, this change does not configure credentials; it only provides `git`.
- If the network/proxy fails, the build can still fail independently of this Dockerfile fix.

## Compatibility Notes
- Backwards compatibility: No application behavior change.
- Data/migrations: None.
- Config/flags: None.

## API/UX Decisions (if applicable)
- Not applicable.
