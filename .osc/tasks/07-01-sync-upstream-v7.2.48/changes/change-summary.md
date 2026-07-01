# Change Summary: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-01
- Owner(s): hewei
- Status: Planning (closure to be filled after implementation)
- Related: `proposal.md`, `spec.md`, `tasks.md`, `regression-checklist.md`, `rollback-notes.md`
- Upstream target: `upstream/main` @ `956ce7cf` (tag `v7.2.48`)
- Last sync point: `21fad9db` (2026-05-21, task `05-22-merge-upstream-non-docker-changes`)
- Pre-sync local `main`: `91dea825`

## What this change will do (planned)

- Absorb `350` upstream commits (`21fad9db..upstream/main`, `664` files) bringing the fork to `v7.2.48`:
  - New subsystems: plugin system (`internal/pluginhost`, `internal/pluginstore`, `sdk/pluginabi`, `sdk/pluginapi`, `sdk/pluginhost`, `sdk/pluginstore`, `examples/plugin/**`), request signature validation (`internal/signature`), safemode (`internal/safemode`), home plugins (`internal/homeplugins`), HTML sanitization (`internal/htmlsanitize`), HTTP fetch helper (`internal/httpfetch`), `cmd/fetch_codex_models`.
  - Translator/runtime/registry/auth/management/SDK updates: `gpt-image-1.5` + direct image API proxy, video URL/auth handling, `disable-cooling` OpenAI compat config, `max` reasoning depth + `service_tiers` for Codex client models, `ResetQuota` endpoint, Codex WS↔SSE full transcript replay, Gemini/Claude/Antigravity fixes, model registry updates (Claude Sonnet 5, Gemini 3.5 Flash variants, deprecated removals), `rebuild_mid_system_message` config, per-auth OAuth model alias, persistent cooldown state.
- Preserve local deployment/ops and CPA-Manager customizations on protected paths (`.github/**`, `Dockerfile`, `docker-*`, `deploy/**`, `.goreleaser.yml`) and in `internal/config/config.go` / `internal/managementasset/updater.go`.
- Re-apply local behavior patches onto upstream's newer structure: Codex invalidated OAuth token failover, OpenAI compat `xhigh` thinking defaults, OpenAI stream null-usage chunks, DeepSeek models + reasoning echo, GPT-5.5 Codex free-tier filter, `host.docker.internal` gateway mapping, websocket body-log cap, string-form system prompt.

## Why

The fork needs upstream's latest functional code (plugin system, security/safemode, image/video, registry/auth/runtime fixes) while keeping local production deployment and CPA-Manager integration stable, following the established sync precedent.

## Notable decisions

- Conflict priority: protected path → local; CPA-Manager defaults → local; local behavior patches → local adapted to upstream structure; everything else → upstream.
- `internal/translator/**` is in scope because the sync spans broader protocol/runtime changes (same precedent as `05-22-merge-upstream-non-docker-changes`).
- Patch-apply with three-way fallback (not plain `git merge`) to avoid upstream deleting local `.osc` state.
- Sync lands on `main`; remote push and production deployment are separate operations.

## Validation (to be filled after implementation)

- (pending) `go build -o test-output ./cmd/server && rm test-output`
- (pending) `go test ./...`
- (pending) focused local-behavior tests
- (pending) protected-path diff empty
- (pending) CPA-Manager defaults present
