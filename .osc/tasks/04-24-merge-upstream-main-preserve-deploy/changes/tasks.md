# Tasks: Merge Upstream Main While Preserving Deployment Files

- Date: 2026-04-24
- Owner(s): hewei
- Related: `spec.md`, `proposal.md`

## Assumptions

- Protected deployment/workflow paths are `.github/**`, `Dockerfile`, `.dockerignore`, `docker-compose.yml`, `docker-compose.*.yml`, and `deploy/**`.
- Upstream functional behavior should win where no protected-path rule applies.
- The merge should stay local; pushing to remotes is a separate step.

## Checklist

- [x] 1) Prepare merge
  - Target: git refs and protected path list
  - Change: confirm current branch, fetch `origin/main` and `upstream/main`, snapshot protected paths implicitly via `HEAD`
  - Verify: `git status --short --branch`

- [x] 2) Merge upstream without committing
  - Target: repository functional code
  - Change: run `git merge --no-commit upstream/main`
  - Verify: inspect conflicts with `git status --short`

- [x] 3) Restore protected paths
  - Target: `.github/**`, Docker/compose/deploy paths
  - Change: restore protected paths from `HEAD`
  - Verify: `git diff HEAD -- <protected paths>`

- [x] 4) Resolve functional conflicts
  - Target: `cmd`, `internal`, `sdk`, `test`, `go.mod`, `go.sum`, `config.example.yaml`
  - Change: resolve conflicts, generally favoring upstream functional code
  - Verify: no unmerged paths remain

- [x] 5) Verify
  - Target: whole Go server and focused packages
  - Change: run build/tests and diff checks
  - Verify: `go build -o test-output ./cmd/server`, focused `go test` commands

- [x] 6) Close workflow artifacts
  - Target: `.osc/tasks/04-24-merge-upstream-main-preserve-deploy/changes`, `.osc/quality-gate.md`
  - Change: record summary, regression checklist, rollback notes, and quality gate
  - Verify: closure files exist

## Notes

- User explicitly said Qwen/IFlow do not need to be preserved.
