# Tasks: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-01
- Owner(s): hewei
- Related: `spec.md`, `proposal.md`

## Assumptions

- Target upstream: `upstream/main` @ `956ce7cf` (tag `v7.2.48`), fetched.
- Last sync point: `21fad9db` (task `05-22-merge-upstream-non-docker-changes`).
- Protected paths (exclude from upstream patch): `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.sh`, `docker-build.ps1`, `docker-compose*.yml`, `.env.cluster.example`, `deploy/**`, `.goreleaser.yml`.
- CPA-Manager defaults live in `internal/config/config.go` and `internal/managementasset/updater.go`.
- Sync stays local; pushing/deploying is separate.

## Checklist

- [ ] 1) Prepare sync
  - Target: git refs and protected path list
  - Change: confirm branch `main` clean, `git fetch upstream`, record pre-sync `HEAD` as rollback point, list protected paths
  - Verify: `git status --short --branch`; `git log --oneline -1 upstream/main` == `956ce7cf`

- [ ] 2) Apply upstream non-protected patch on a temporary branch
  - Target: repository functional code (everything except protected paths)
  - Change: create `task/sync-upstream-v7.2.48`; generate `21fad9db..upstream/main` patch excluding protected paths; apply with three-way fallback (`git apply --3way` or `git checkout upstream/main -- <path>` for clean additions)
  - Verify: `git status --short` shows expected added/modified paths; protected paths untouched

- [ ] 3) Restore protected paths
  - Target: `.github/**`, Docker/compose/deploy, `.goreleaser.yml`
  - Change: restore protected paths from pre-sync `HEAD`
  - Verify: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml` is empty

- [ ] 4) Re-assert CPA-Manager defaults
  - Target: `internal/config/config.go`, `internal/managementasset/updater.go`
  - Change: ensure `DefaultPanelGitHubRepository = "https://github.com/seakee/CPA-Manager"` and default release/fallback URLs point at `seakee/CPA-Manager`
  - Verify: `git grep -n -i 'CPA-Manager' -- internal/config/config.go internal/managementasset/updater.go`

- [ ] 5) Re-apply local behavior patches onto upstream structure
  - Target: Codex invalidated OAuth failover, OpenAI compat xhigh thinking defaults, OpenAI stream null-usage chunks, DeepSeek models + reasoning echo, GPT-5.5 Codex free-tier filter, `host.docker.internal` gateway mapping, websocket body-log cap, string-form system prompt
  - Change: port each patch to upstream's new call sites/structure; fix compilation
  - Verify: focused tests listed in regression-checklist pass

- [ ] 6) Resolve remaining conflicts and format
  - Target: any conflicted Go source/tests/imports
  - Change: resolve conflicts (priority per spec); `gofmt -w` changed Go files
  - Verify: `git diff --name-only --diff-filter=U` empty; `git diff --check`; `gofmt -l` clean

- [ ] 7) Run quality gates
  - Target: full Go repository
  - Change: run build/tests; fix failures introduced by the sync
  - Verify: `go build -o test-output ./cmd/server && rm test-output`; `go test ./...`; focused tests

- [ ] 8) Record closure artifacts
  - Target: `.osc/tasks/07-01-sync-upstream-v7.2.48/changes/` and `.osc/quality-gate.md`
  - Change: write change-summary, regression-checklist (executed), rollback-notes, quality-gate results
  - Verify: closure files exist and reflect actual command results

## Notes

- Upstream added ~664 changed files since `21fad9db`; the bulk lands under `internal/` (translator, runtime, api, pluginhost, watcher, signature, pluginstore) and `sdk/`, plus new `examples/plugin/**` and `cmd/fetch_codex_models`.
- Module path is already `v7` on both branches (prior sync handled the v6→v7 bump), so no module-path migration this round.
- Precedent for protected-path exclusion and patch-apply approach: `05-22-merge-upstream-non-docker-changes` and `04-24-merge-upstream-main-preserve-deploy`.
