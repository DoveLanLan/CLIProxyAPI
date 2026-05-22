# Spec: Merge Upstream Non-Docker Changes

- Date: 2026-05-22
- Owner(s): hewei / Codex
- Related: proposal.md, tasks.md

## Repo Snapshot

- Modules/components: `cmd/server` binary, backend/runtime packages under `internal/`, reusable SDK under `sdk/`, docs/examples/tests at top level.
- Toolchains: Go module; format with `gofmt`; test with `go test ./...`; build with `go build -o test-output ./cmd/server && rm test-output`.
- Quality/CI: local repo requires artifact-first `.osc` workflow; PR build gate compiles `./cmd/server`; `.github` and Docker-related upstream changes are out of scope for this task.
- Confidence: High.
- Evidence: `AGENTS.md`, `.osc/workflow.md`, `.osc/spec/project-spec.md`, `go.mod`, `config.example.yaml`.

## Scope

### In scope

- Apply upstream changes from `upstream/main` since the merge base with the current `HEAD`.
- Include Go source, tests, registry data, README/docs, examples, assets, module files, and non-Docker configuration templates.
- Include `internal/translator/**` changes only as part of the broader upstream sync.

### Out of scope

- `.github/**`.
- `Dockerfile`, `.dockerignore`, `docker-build.sh`, `docker-build.ps1`, `docker-compose*.yml`, and `.env.cluster.example`.
- Local `.osc` workflow/task history outside this task's own change artifacts.
- Local `config.yaml`, `auths/`, and `logs/` data.

## Acceptance Criteria (testable)

1. Upstream non-excluded patch is applied to the current working tree. (Verify: `git diff --name-only` and spot-check upstream-added packages/files.)
2. Excluded `.github` and Docker-related files are not modified by the merge. (Verify: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example`.)
3. The repository is formatted. (Verify: `gofmt -w` on changed Go files.)
4. Full test suite passes. (Verify: `go test ./...`.)
5. Server compile gate passes. (Verify: `go build -o test-output ./cmd/server && rm test-output`.)

## Behavior / Requirements

The final working tree should contain upstream functionality for Home control plane support, xAI support, Codex client model handling, updated reasoning/thinking metadata, Redis queue protocol handling, logging/usage improvements, and protocol translator fixes. Docker and GitHub workflow changes must remain absent from the applied patch.

## Edge Cases

- If an upstream patch hunk conflicts with local code, resolve manually while preserving local fork behavior unless upstream's change is required for compile/test compatibility.
- If excluding Docker files leaves a new non-Docker reference to a missing Docker artifact, keep the reference only if it is documentation-only and does not break tests/build.
- If generated/large registry JSON changes conflict, prefer valid upstream data and run tests that parse registry definitions.

## Compatibility Notes

- Backwards compatibility: existing local config and auth files must continue to load.
- Data/migrations: no database schema migration is expected.
- Config/flags: upstream may add config fields; defaults must keep old configs valid.

## API/UX Decisions

- Inputs/outputs: no deliberate local API redesign; imported upstream behavior should match upstream.
- States/errors: keep upstream error handling unless it conflicts with local user-facing behavior.
- Telemetry/logging: no secrets should be logged; upstream logging changes must preserve masking behavior.
