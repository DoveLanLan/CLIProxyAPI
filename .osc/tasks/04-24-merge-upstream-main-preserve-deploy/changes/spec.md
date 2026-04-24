# Spec: Merge Upstream Main While Preserving Deployment Files

- Date: 2026-04-24
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## Repo Snapshot (from step 0)

- Modules/components: Go server entrypoint in `cmd/server`; server internals in `internal/*`; embeddable SDK and handlers in `sdk/*`; regression tests in package and top-level `test/`. Confidence: High. Evidence: `.osc/spec/project-spec.md`, `go.mod`, `cmd/server`, `internal`, `sdk`.
- Toolchains: Go module build; minimum CI gate is `go build ./cmd/server`; focused package tests are used for touched packages. Confidence: High. Evidence: `.github/workflows/pr-test-build.yml`, `.osc/spec/project-spec.md`.
- Quality/CI: PR path guard protects `internal/translator/**` in upstream workflow, but this merge intentionally imports upstream translator changes. Deployment/workflow files are protected by user request. Confidence: High. Evidence: `.osc/spec/project-spec.md`, user request.

## Scope

### In scope

- Merge functional code from `upstream/main` into local `main`.
- Accept upstream removal of Qwen and iFlow.
- Accept upstream model registry, auth, Codex, Responses, Antigravity, Claude, management API, SDK, and translator changes.
- Restore local workflow and deployment paths after merge.

### Out of scope

- `.github/**` changes from upstream.
- Docker/deployment path changes from upstream.
- Production config file changes outside tracked template/config example changes.
- Remote push or production deployment.

## Acceptance Criteria (testable)

1. `git status` shows no unresolved merge conflicts. (Verify: `git diff --check` and `git status --short`)
2. Protected paths match pre-merge `HEAD`. (Verify: `git diff HEAD -- .github Dockerfile docker-compose.yml docker-compose.*.yml .dockerignore deploy` is empty or only expected local task artifacts are outside this set)
3. Qwen and iFlow are not restored from the fork after merge. (Verify: removed upstream paths remain absent)
4. Server builds after the merge. (Verify: `go build -o test-output ./cmd/server`)
5. Focused tests for high-risk changed areas run where practical. (Verify: selected `go test` commands)

## Behavior / Requirements

- Prefer upstream functional code for conflicts unless it would modify protected deployment/workflow paths.
- For protected paths, keep the fork's current content exactly.
- Keep local `.osc/` task artifacts for auditability.
- Do not introduce secrets or environment-specific config.

## Edge Cases

- Conflicts in `config.example.yaml` must keep upstream functional configuration options unless they are purely deployment instructions.
- Conflicts caused by upstream deleting Qwen/iFlow should resolve to deletion.
- Conflicts in model registry should resolve to upstream's JSON catalog/updater model.
- Large translator conflicts should resolve to upstream behavior because this merge intends to catch up with upstream functionality.

## Compatibility Notes

- Backwards compatibility: Qwen and iFlow compatibility is intentionally dropped per user direction.
- Data/migrations: No database schema migration is expected.
- Config/flags: Upstream config fields may be added; tracked production deployment files remain local.

## API/UX Decisions (if applicable)

- Inputs/outputs: upstream API route additions are accepted, including health and image routes.
- States/errors: upstream auth scheduling, retry, and stream behavior are accepted.
- Telemetry/logging: upstream logging changes are accepted outside protected deployment files.
