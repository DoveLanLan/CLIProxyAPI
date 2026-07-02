# Change Summary: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-01
- Owner(s): hewei
- Status: Implemented and verified locally
- Related: `proposal.md`, `spec.md`, `tasks.md`, `regression-checklist.md`, `rollback-notes.md`
- Upstream target: `upstream/main` @ `956ce7cf` (tag `v7.2.48`)
- Last sync point: `21fad9db` (2026-05-21, task `05-22-merge-upstream-non-docker-changes`)
- Pre-sync local implementation branch base: `307f5ff8`

## What changed

- Absorbed upstream functional code through `v7.2.48`, including:
  - Plugin system: `internal/pluginhost`, `internal/pluginstore`, `sdk/pluginabi`, `sdk/pluginapi`, `sdk/pluginhost`, `sdk/pluginstore`, and `examples/plugin/**`.
  - New support modules: `internal/signature`, `internal/safemode`, `internal/homeplugins`, `internal/htmlsanitize`, `internal/httpfetch`, `cmd/fetch_codex_models`.
  - Runtime/protocol updates: `gpt-image-1.5`, direct image API proxy, video URL/auth handling, `disable-cooling`, `max` reasoning depth, `service_tiers`, `ResetQuota`, Codex WS↔SSE replay, Claude/Gemini/Antigravity/xAI fixes, registry updates including Claude Sonnet 5.
- Followed upstream removals for Amp and Gemini CLI code paths.
- Preserved local protected deployment/ops paths and kept upstream CI/Docker/release changes out; only intentional protected-path edit is `deploy/README.md` updating the panel source to Plus.
- Changed built-in management panel defaults/docs to `seakee/CPA-Manager-Plus`:
  - `internal/config/config.go`
  - `internal/managementasset/updater.go`
  - `config.example.yaml`
  - `README.md`, `README_CN.md`, `README_JA.md`
  - `deploy/README.md`
  - `docs/handoff/cpa-manager-fork-panel-release.md`
- Re-ported local behavior patches onto upstream v7.2.48:
  - Codex invalidated OAuth token disables bad auth and falls through to the next auth even when `max-retry-credentials=1`.
  - OpenAI-compatible default thinking includes `none/low/medium/high/xhigh` and zero budget.
  - OpenAI stream usage parser ignores `usage:null` and supports Responses usage fields.
  - String-form Claude system prompts are preserved.
  - WebSocket request/response body log growth is capped.
  - DeepSeek static model definitions and reasoning echo support remain available.
  - GPT-5.5 remains present for Codex team/plus/pro/static lookup and excluded from Codex free.
  - xAI thinking efforts normalize `minimal→low` and `xhigh/max→high` for xAI Responses output.

## Why

The fork needed upstream's latest functional code (plugin system, safemode/signature, image/video, registry/auth/runtime/translator fixes) while keeping local production deployment and CPA-Manager-Plus integration stable.

## Notable decisions

- Conflict priority used: protected path → local; CPA-Manager-Plus defaults → local; local behavior patches → local behavior adapted to upstream structure; everything else → upstream.
- `internal/translator/**` changes are part of this broad upstream sync, not a translator-only task.
- Upstream-deleted Amp and Gemini CLI paths were removed instead of preserved.
- Remote push and production deployment remain separate operations.

## Validation

- PASS: `git diff --name-only --diff-filter=U` = 0.
- PASS: conflict-marker scan over Go/YAML/Markdown files.
- PASS: `git diff --cached --check`.
- PASS: `go build -o test-output ./cmd/server && rm test-output`.
- PASS: `go test ./...`.
- PASS: focused local-behavior tests for Codex invalidated OAuth failover, OpenAI compat xhigh thinking, null usage parsing, string system prompt, websocket log cap, registry/GPT-5.5, management usage, redisqueue, and OpenAI image/video handlers.
- PASS: CPA-Manager-Plus defaults verified in code/config/docs.

---

## Addendum: Remove Legacy CPA-Manager from Production Deploy

- Date: 2026-07-02
- Status: Implemented and verified locally

### What changed

- Removed the legacy `cpa-manager` service from `deploy/compose.production.yml`, including its host port binding, environment, volume, and dependency on `cli-proxy-api`.
- Removed stale production env sample entries for the retired service from `deploy/.env.example`.
- Deleted `.github/workflows/update-cpa-manager-image.yml`, which only updated/restarted the obsolete service.
- Updated `deploy/README.md` so production management is documented through the CLIProxyAPI private management URL and CPA Manager Plus panel source.
- Updated `.osc/spec/project-spec.md` and this task's change artifacts for the deployment hotfix.

### Why

GitHub Actions failed during `Deploy production stack` because the old `cpa-manager` service attempted to bind `100.67.99.9:18318`, but that port is already allocated by the current CPA Manager Plus setup. The production stack should no longer manage or start the legacy standalone CPA-Manager container.

### Notable decisions

- Kept `deploy/scripts/remote-deploy.sh` unchanged because it already runs `docker compose up -d --remove-orphans`; after the updated compose file is installed, the old compose-managed service should be removed as an orphan.
- Left local development `docker-compose.yml` and `docker-compose.example.yml` unchanged because this hotfix is scoped to production deploy.
- Did not remove any remote `data/cpa-manager/` data; stale data can be backed up or removed manually on the VPS if needed.

### Validation

- PASS: stale production reference scan returned `no stale legacy CPA-Manager production refs`.
- PASS: `.github/workflows/update-cpa-manager-image.yml` is absent.
- PASS: `docker compose -f deploy/compose.production.yml --env-file deploy/.env.example config --services` returned `cli-proxy-api`.
- PASS: `docker compose -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml --env-file deploy/.env.example config --services` returned `split-proxy` and `cli-proxy-api`.
- PASS: `go build -o test-output ./cmd/server && rm test-output`.
