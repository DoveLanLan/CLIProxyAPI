# Proposal: Sync Upstream v8.0.43 Preserving Deploy, CPA-Manager and Local Providers

- Date: 2026-10-26
- Owner(s): hewei
- Status: Accepted
- Upstream target: `upstream/main` @ tag `v8.0.43`
- Last sync point: `e92bd900` (2026-07-21, merged upstream `01f387f4` = `v7.2.93`, task `07-01-sync-upstream-v7.2.48`)
- New upstream commits to absorb: `1441` (`v7.2.93..upstream/main`), `1810` files changed
- Local-only commits: `94` (`upstream/main..HEAD`), `552` files touched since merge-base
- Overlapping files: `52`; `git merge-tree` content conflicts: `34`

## Context / Problem

The fork has not synced with `router-for-me/CLIProxyAPI` since `e92bd900` (upstream `v7.2.93`, 2026-07-21). Upstream has since:

1. **Moved to a new major version `v8`** (`31f4cfab feat: migrate CLIProxyAPI to version 8`), rewriting Go import paths in 912 files (`.../v7/...` → `.../v8/...`), bumping `go.mod` to `go 1.26.10`, and adding WebRTC/Pion, zeroconf and regexp2 dependencies.
2. **Heavily reworked Antigravity** (797 commits touch antigravity paths): executor split into `_auth/_credits/_execute/_request/_stream/_tokens`, new `internal/signature/claude_antigravity_validation.go`, `internal/translator/common/antigravity_tools.go`, `internal/cache/antigravity_interactions_session.go`, `helps/antigravity_{compaction,connection_trace,interactions_state}.go`, `sdk/cliproxy/antigravity_model_probes.go`; conversation compaction + capsule encryption, Claude 5.5 CAQS signature replay, per-account model entitlements, web search translation, `video_url`, sensitive-word obfuscation.
3. Introduced Home/session-affinity subsystems (Merkle LCP session affinity, session tree, credential concurrency, CAS replay), a Devin/Cognition provider, pluginhost payload rules, httptransport decorators and a payload finalizer.

The fork carries local-only work that must survive:

- **Deployment/ops (protected)**: `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.sh`, `docker-build.ps1`, `docker-compose*.yml`, `.env.cluster.example`, `deploy/**`, `.goreleaser.yml`.
- **CPA-Manager**: `DefaultPanelGitHubRepository = "https://github.com/seakee/CPA-Manager-Plus"` in `internal/config/config.go`, release/fallback URLs in `internal/managementasset/updater.go`, panel refs in `config.example.yaml`.
- **Local providers / protocol patches**:
  - xAI dedicated rotating proxy pool + Resin (`internal/api/handlers/management/xai_proxy_{pool,subscriptions}.go`, `internal/config/*xai*`, `internal/runtime/executor/{xai_proxy_pool_executor,xai_resin_proxy_executor_test,helps/xai_proxy_pool,helps/xai_resin_proxy*}.go`, `internal/auth/xai/**`, `internal/thinking/provider/xai/**`).
  - Command Code provider (`internal/runtime/executor/commandcode_executor.go`, `helps/commandcode_*.go`, `sdk/cliproxy/{commandcode_models.go,auth/commandcode.go}`).
  - DeepSeek models + reasoning echo (`helps/openai_compat_deepseek.go`, registry overrides).
  - OpenAI/DeepSeek vision preflight and tool-result image relocation (`helps/openai_tool_images.go`, `helps/openai_message_links.go`).
  - OpenAI compat `xhigh` thinking defaults; OpenAI stream null-usage chunk handling.
  - Codex OAuth invalidated-token failover; GPT-5.5 Codex support; websocket body-log growth cap; string-form system prompt preservation.
  - Claude upstream error metadata retention; per-model thinking override for `claude-api-key` models.
  - `host.docker.internal` gateway mapping.

## Goals (Why/What)

- Bring local `main` functionally to upstream `v8.0.43`.
- Complete the module path migration `/v7` → `/v8` for **local-only** files that upstream's rename commit cannot reach.
- Keep every protected deployment/CI/Docker path byte-identical to local `HEAD`.
- Keep CPA-Manager defaults pointing at `seakee/CPA-Manager-Plus`.
- Re-apply each local provider/protocol patch onto upstream v8's structure; drop local patches that upstream has independently fixed (adopt upstream).
- Keep the repo buildable and tests green.

## Constraints

- Protected paths (never take upstream): `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.sh`, `docker-build.ps1`, `docker-compose*.yml`, `.env.cluster.example`, `deploy/**`, `.goreleaser.yml`, `.osc/**`, `.claude/**`.
- CPA-Manager defaults in `internal/config/config.go` / `internal/managementasset/updater.go` stay local.
- `internal/translator/**` is normally protected by CI for ordinary PRs; it is in scope here because the sync spans broader protocol/runtime changes (same precedent as `05-22-merge-upstream-non-docker-changes` and `07-01-sync-upstream-v7.2.48`).
- `internal/runtime/executor/` holds executors + their unit tests only; helper files go under `helps/` (AGENTS.md).
- No secrets or environment-specific config committed; `config.yaml` stays ignored.
- No `log.Fatal`/`log.Fatalf`; logrus structured logging (AGENTS.md).
- Stay local — pushing to remotes and production deployment are separate operations.
- Do not create a merge commit on `main` until the branch is verified; work happens on `task/sync-upstream-v8`.

## Non-goals

- Do not redesign upstream changes during the sync.
- Do not adopt upstream's deployment/CI/Docker changes (explicit exclusion).
- Do not change local runtime config files such as `config.yaml`, `auths/`, `logs/`.
- Do not remove local providers (xAI pool, Command Code, DeepSeek overrides) even if upstream refactored the same call sites.

## Proposed Approach

1. Branch `task/sync-upstream-v8` from local `main` (`cf5070da`).
2. `git merge upstream/main` (34 content conflicts expected) rather than patch-apply: the fork's local work is mostly *new files* upstream never touched (only 52 overlapping files), so a real merge preserves them and keeps history/bisectability.
3. Resolve conflicts with this priority:
   - Protected path → keep local `HEAD`.
   - CPA-Manager default → keep local value.
   - Local provider/protocol patch → keep local behavior, adapt to upstream v8 structure.
   - Everything else (new subsystems, antigravity/translator/runtime/registry/auth/management/SDK) → take upstream.
4. Rewrite `.../CLIProxyAPI/v7` → `.../CLIProxyAPI/v8` in the 24 local-only Go files (upstream's rename commit covers the other ~475).
5. Restore protected paths from pre-sync `HEAD` where the merge modified them.
6. Run quality gates: `gofmt`, `go build ./cmd/server`, `go vet`, `go test ./...`, focused provider tests.

## Conflict resolution priority (summary)

| Category | Decision |
|---|---|
| `.github/**`, Docker, compose, `deploy/**`, `.goreleaser.yml` | keep local |
| CPA-Manager defaults | keep local |
| xAI pool / Resin / Command Code / DeepSeek / thinking overrides | keep local, adapt to v8 |
| Upstream new subsystems + antigravity/translator/runtime/registry/auth/management/SDK | take upstream |
| Case-by-case (local patch upstream also fixed) | take upstream, delete local patch |

## Risks

- **R1 — module path half-migration**: upstream renamed 912 files; any local-only file left on `/v7` breaks compile. Mitigation: grep gate `grep -rl CLIProxyAPI/v7 --include='*.go'` must return 0.
- **R2 — Go toolchain**: `go.mod` requires `1.26.10`; local toolchain is `1.26.0` with `GOTOOLCHAIN=auto`, so a 1.26.10 toolchain must download. Mitigation: verify `go build` succeeds early; if network is restricted, pin down a fallback.
- **R3 — xAI/Command Code executors drift**: upstream may have restructured `openai_compat_executor.go`, `xai_executor.go`, `codex_websockets_executor.go`. Mitigation: focused tests per provider + `go build`.
- **R4 — Antigravity local patch superseded**: `9e5bf4d5 fix(antigravity): accept array tool schemas without items` is covered by upstream `b5cde4ba`. Mitigation: adopt upstream, drop local patch, verify `gemini_schema_test.go` still covers the case.
- **R5 — CPA-Manager URL clobbered**: upstream may touch `managementasset/updater.go`. Mitigation: explicit grep gate on `seakee/CPA-Manager-Plus`.

## Rollback

Local `main` stays at `cf5070da` throughout. Rollback = `git checkout main` (and `git branch -D task/sync-upstream-v8`). See `changes/rollback-notes.md`.
