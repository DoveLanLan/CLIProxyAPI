# Tasks: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-01
- Owner(s): hewei
- Related: `spec.md`, `proposal.md`

## Assumptions

- Target upstream: `upstream/main` @ `956ce7cf` (tag `v7.2.48`), fetched.
- Last sync point: `21fad9db` (task `05-22-merge-upstream-non-docker-changes`).
- Protected paths (exclude from upstream patch): `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.sh`, `docker-build.ps1`, `docker-compose*.yml`, `.env.cluster.example`, `deploy/**`, `.goreleaser.yml`.
- CPA-Manager defaults live in `internal/config/config.go` and `internal/managementasset/updater.go` and point to `seakee/CPA-Manager-Plus`.
- Sync stays local; pushing/deploying is separate.

## Checklist

- [x] 1) Prepare sync
  - Target: git refs and protected path list
  - Change: confirmed branch state, fetched/used `upstream/main`, recorded pre-sync rollback point `307f5ff8` for implementation branch (pre-plan-main was `91dea825`).
  - Verify: `git status --short --branch`; `git log --oneline -1 upstream/main` == `956ce7cf`.

- [x] 2) Apply upstream non-protected patch on a temporary branch
  - Target: repository functional code.
  - Change: created `task/sync-upstream-v7.2.48`, merged upstream with conflict resolution, accepted upstream functional code and removed upstream-deleted Amp/Gemini CLI paths.
  - Verify: `git status --short` shows expected large upstream sync; no unresolved paths remain.

- [x] 3) Restore protected paths
  - Target: `.github/**`, Docker/compose/deploy, `.goreleaser.yml`.
  - Change: restored protected CI/Docker/release paths from pre-sync state; kept only intentional `deploy/README.md` CPA-Manager-Plus documentation update.
  - Verify: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml` shows only `deploy/README.md`.

- [x] 4) Re-assert CPA-Manager defaults
  - Target: `internal/config/config.go`, `internal/managementasset/updater.go`.
  - Change: set `DefaultPanelGitHubRepository`, release API URL, fallback download URL, examples, READMEs, and deploy handoff docs to `https://github.com/seakee/CPA-Manager-Plus`.
  - Verify: `git grep -n 'seakee/CPA-Manager-Plus' -- internal/config/config.go internal/managementasset/updater.go config.example.yaml README.md README_CN.md README_JA.md deploy/README.md`.

- [x] 5) Re-apply local behavior patches onto upstream structure
  - Target: Codex invalidated OAuth failover, OpenAI compat xhigh thinking defaults, OpenAI stream null-usage chunks, DeepSeek models + reasoning echo, GPT-5.5 Codex free-tier filter, websocket body-log cap, string-form system prompt, xAI effort normalization.
  - Change: ported local patches to upstream v7.2.48 structure; fixed tests/fixtures impacted by upstream executor and thinking changes.
  - Verify: focused tests listed in regression-checklist passed.

- [x] 6) Resolve remaining conflicts and format
  - Target: conflicted Go source/tests/imports.
  - Change: resolved all conflicts; formatted changed Go files with `gofmt`.
  - Verify: `git diff --name-only --diff-filter=U` = 0; `git diff --cached --check` passed; no conflict markers found.

- [x] 7) Run quality gates
  - Target: full Go repository.
  - Change: ran full test/build and fixed failures introduced by the sync.
  - Verify: `go build -o test-output ./cmd/server && rm test-output`; `go test ./...`; focused tests all passed.

- [x] 8) Record closure artifacts
  - Target: `.osc/tasks/07-01-sync-upstream-v7.2.48/changes/` and `.osc/quality-gate.md`.
  - Change: updated tasks, change summary, regression checklist, and quality gate.
  - Verify: closure files exist and reflect actual command results.

## Notes

- Upstream added ~664 changed files since `21fad9db`; the bulk lands under `internal/` (translator, runtime, api, pluginhost, watcher, signature, pluginstore) and `sdk/`, plus new `examples/plugin/**` and `cmd/fetch_codex_models`.
- Module path is already `v7` on both branches (prior sync handled the v6→v7 bump), so no module-path migration this round.
- Upstream removed Amp and Gemini CLI paths; this sync follows upstream removal.
- `CPA-Manager-Plus` is the current built-in panel source; `seakee/CPA-Manager-Plus` latest release was verified to include `management.html`.
