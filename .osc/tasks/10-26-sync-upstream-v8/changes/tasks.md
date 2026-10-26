# Tasks: Sync Upstream v8.0.43

- Date: 2026-10-26
- Owner(s): hewei
- Related: `spec.md`, `proposal.md`

## Assumptions

- Target upstream: `upstream/main` @ `v8.0.43`, fetched.
- Last sync point: `e92bd900` (upstream `01f387f4` = `v7.2.93`).
- Pre-sync rollback point: `cf5070da`.
- Protected paths: `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.*`, `docker-compose*.yml`, `.env.cluster.example`, `deploy/**`, `.goreleaser.yml`, `.osc/**`, `.claude/**`.
- CPA-Manager defaults live in `internal/config/config.go` and `internal/managementasset/updater.go` → `seakee/CPA-Manager-Plus`.
- Sync stays local; pushing/deploying is separate.

## Checklist

- [ ] 1) Create branch and merge upstream
  - Target: `task/sync-upstream-v8`
  - Change: branch from `cf5070da`, `git merge upstream/main`.
  - Verify: `git rev-parse --abbrev-ref HEAD` == `task/sync-upstream-v8`; `git log -1 --oneline upstream/main` == `v8.0.43`; conflict list matches the 34 expected files.

- [ ] 2) Resolve the 34 content conflicts
  - Target: conflicted files (see spec §7 rules).
  - Change: per-file resolution by category.
  - Verify: `git diff --name-only --diff-filter=U` empty; `grep -rn '^<<<<<<<' --include='*.go' .` empty.

- [ ] 3) Protected paths restored
  - Target: `.github/**`, Docker/compose/deploy, `.goreleaser.yml`, `.osc/**`.
  - Change: `git checkout cf5070da -- <protected paths>` where the merge changed them.
  - Verify: Gate G2 — `git diff --stat cf5070da -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml` shows nothing.

- [ ] 4) CPA-Manager defaults re-asserted
  - Target: `internal/config/config.go`, `internal/managementasset/updater.go`, `config.example.yaml`.
  - Change: keep `seakee/CPA-Manager-Plus` URLs.
  - Verify: Gate G3 grep.

- [ ] 5) Module path `/v7` → `/v8` for local-only files
  - Target: the 24 local-only Go files still importing `/v7`.
  - Change: rewrite import paths to `/v8`.
  - Verify: Gate G1 — `grep -rl 'CLIProxyAPI/v7' --include='*.go' .` returns 0.

- [ ] 6) Re-apply local provider/protocol patches onto v8 structure
  - Target: xAI pool/Resin, Command Code, DeepSeek, vision preflight, thinking overrides, Codex failover, Claude error metadata.
  - Change: port to upstream v8 call sites; delete superseded patches (antigravity array schema → upstream `b5cde4ba`).
  - Verify: focused provider tests in `regression-checklist.md` pass.

- [ ] 7) Toolchain + deps
  - Target: `go.mod` / `go.sum`, Go `1.26.10`.
  - Change: `go mod tidy` if needed; ensure 1.26.10 toolchain resolves.
  - Verify: `go version` inside module context; `go build` proceeds.

- [ ] 8) Format and static checks
  - Target: changed Go files.
  - Change: `gofmt -w` on changed files.
  - Verify: `gofmt -l` clean; `go vet ./...`.

- [ ] 9) Build + test
  - Target: whole repo.
  - Change: fix failures introduced by the sync.
  - Verify: `go build -o test-output ./cmd/server && rm test-output`; `go test ./...`.

- [ ] 10) Closure artifacts
  - Target: `.osc/tasks/10-26-sync-upstream-v8/changes/` + `.osc/quality-gate.md`.
  - Change: write `change-summary.md`, `regression-checklist.md`, `rollback-notes.md`; update `task.json` status.
  - Verify: files exist and record real command output.

## Notes

- The fork's local work is mostly *new files* upstream never touched (52 overlapping files, 34 conflicts), so a real `git merge` is cheaper and more auditable than the patch-apply approach used in `07-01-sync-upstream-v7.2.48`.
- Upstream `31f4cfab` already renamed ~475 of our shared Go files to `/v8`; only 24 purely-local files need manual rewriting.
- Upstream v8 adds WebRTC/Pion + zeroconf deps; `go.sum` will grow substantially.
- `go 1.26.10` vs local `go 1.26.0`: `GOTOOLCHAIN=auto` will fetch the patch toolchain.
