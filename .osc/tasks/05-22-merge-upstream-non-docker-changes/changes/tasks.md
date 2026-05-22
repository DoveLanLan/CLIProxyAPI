# Tasks: Merge Upstream Non-Docker Changes

- Date: 2026-05-22
- Owner(s): hewei / Codex
- Related: proposal.md, spec.md

## Assumptions

- Target upstream branch is `upstream/main` from `https://github.com/router-for-me/CLIProxyAPI`.
- Docker-related exclusions are `.dockerignore`, `Dockerfile`, `docker-build.*`, `docker-compose*.yml`, and `.env.cluster.example`.
- Existing local `.osc` files are local workflow state and should not be deleted by upstream sync.

## Checklist

- [x] 1) Fetch and inspect upstream
  - Target: Git remotes and branch graph.
  - Change: Fetch `upstream/main`, identify merge base and changed paths.
  - Verify: `git fetch upstream` and `git diff --name-status`.

- [x] 2) Apply upstream non-excluded patch
  - Target: Repository files except excluded `.github` and Docker-related paths.
  - Change: Apply `merge-base..upstream/main` patch with three-way conflict handling.
  - Verify: no unresolved conflict markers and excluded paths unchanged.

- [x] 3) Resolve conflicts and format
  - Target: Any conflicted Go source/tests plus generated imports.
  - Change: Resolve conflicts, run `gofmt` on changed Go files.
  - Verify: `git diff --check` and `gofmt`.

- [x] 4) Run quality gates
  - Target: Full Go repository.
  - Change: Run tests/build and fix failures introduced by the sync.
  - Verify: `go test ./...` and `go build -o test-output ./cmd/server && rm test-output`.

- [x] 5) Record closure artifacts
  - Target: `.osc/tasks/05-22-merge-upstream-non-docker-changes/changes/` and `.osc/quality-gate.md`.
  - Change: Add change summary, regression checklist, rollback notes, and quality-gate results.
  - Verify: files exist and reflect actual command results.

## Notes

- Upstream changed 424 non-excluded paths from the merge base, with most changes under `internal/` and `sdk/`.
- Upstream Docker-only changes detected before exclusion: `Dockerfile`, `docker-build.sh`, `docker-compose.cluster.yml`, and `.env.cluster.example`.
