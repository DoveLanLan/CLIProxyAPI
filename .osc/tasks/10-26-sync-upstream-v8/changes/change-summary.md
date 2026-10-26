# Change Summary: Sync Upstream v8.0.43

- Date: 2026-10-26
- Owner(s): hewei
- Branch: `task/sync-upstream-v8` → merged into `main`
- Pre-sync local HEAD (rollback point): `cf5070da` (`v7.2.93-94-gcf5070da`)
- Upstream target: `upstream/main` @ `v8.0.43`
- Last sync point: `e92bd900` (merged upstream `01f387f4` = `v7.2.93`, 2026-07-21)

## What landed

### 1. Major version migration v7 → v8

- `module github.com/router-for-me/CLIProxyAPI/v8`, `go 1.26.10`.
- Upstream's `31f4cfab` rewrote import paths in ~912 files; the remaining 25 fork-only Go files were rewritten by this sync (Gate G1 = 0 `/v7` references).
- New upstream dependencies carried in: `pion/webrtc`, `pion/ice`, `pion/interceptor`, `pion/rtp`, `pion/sdp`, `pion/stun`, `libp2p/zeroconf`, `go-git v6.0.0-alpha.5`, `tiktoken-go v0.8.1`, `regexp2`.
- Upstream split `internal/config/` into ~70 files and `sdk/cliproxy/auth/conductor.go` into `conductor_{models,execution,stream,lifecycle,cooldown,refresh,selection,home}.go`. Fork code was re-applied onto the new split.

### 2. Antigravity (`797` upstream commits)

Adopted wholesale, including:

- executor split into `_auth` / `_credits` / `_execute` / `_request` / `_stream` / `_tokens`,
- new `internal/signature/claude_antigravity_validation.go`, `internal/translator/common/antigravity_tools.go`, `internal/cache/antigravity_interactions_session.go`, `helps/antigravity_{compaction,connection_trace,interactions_state}.go`, `sdk/cliproxy/antigravity_model_probes.go`,
- conversation compaction + capsule encryption, Claude 5.5 CAQS thinking-signature replay, per-account model entitlements, OpenAI-Responses web search translation, `video_url` parts, sensitive-word obfuscation, short-connection pool hardening, Hub request alignment + versioned model-cache invalidation.

### 3. Fork customizations preserved

| Area | Status | Notes |
|---|---|---|
| Deploy/CI/Docker (`deploy/**`, `.github/**`, `Dockerfile`, compose, `.goreleaser.yml`) | untouched | Gate G2 diff empty |
| CPA-Manager-Plus panel defaults | preserved | `config_defaults.go`, `managementasset/updater.go`, `config.example.yaml` |
| xAI rotating proxy pool + Resin | preserved | management routes re-registered on `/v0/management/xai-proxy-pool/**`; `XAIAutoExecutor` wiring re-applied on v8's 5-arg session-target API |
| Native Command Code provider | preserved | executor/model registration re-applied in `service_executors.go` / `service_models.go`; config field kept |
| DeepSeek models + reasoning echo | preserved | catalog re-added; `openai_compat_executor.go` reasoning-echo + `DetectDeepSeekClaudeCompatibilityIssue` hooks re-applied |
| Tool-result image relocation / DeepSeek vision | preserved | `NormalizeOpenAIToolImages` + `NormalizeOpenAIToolMessageLinks` hooks re-applied |
| OpenAI-compat thinking fallback | preserved | `resolveOpenAICompatThinking` re-applied onto `modelconfig.ResolveModelInfo` pipeline |
| xAI thinking level normalization | preserved | `provider/xai/apply.go` `minimal→low`, `xhigh/max→high` |
| xAI in-stream free-tier 429 failover | preserved | `xaiStreamEventError` + `invalid_grant`→401 |
| Claude upstream error metadata | preserved | `code` / `upstream_model` / `request_id` on `/v1/messages` errors |

### 4. Local patches superseded by upstream (dropped, upstream adopted)

| Local work | Upstream replacement |
|---|---|
| `9e5bf4d5` antigravity array tool schemas without `items` | `b5cde4ba` default `items` schema for Gemini/Antigravity |
| `e22edff1` Codex OAuth invalidated token → disable credential | `feb63984` (issue #6742): `RejectedAccessToken` + background refresh + 96h proactive lead; strictly better (recovers instead of permanently disabling). Local test `TestManager_CodexInvalidatedOAuthTokenDisablesAndFallsBackWithMaxRetryOne` removed. |
| `2b9e4b79` websocket body-log growth cap | upstream removed the websocket body-log accumulator entirely; 3 obsolete tests removed |
| `7632a122` string system prompt preservation | upstream `sdk/api/handlers/handlers.go` already identical |
| `sdk/api/handlers/bootstrap_keepalive.go` (`StartStreamingBootstrapKeepAlive`) | upstream `handlers.NewPreFirstTokenKeepAliveTicker` + `ExecuteStreamWithPreFTKeepAlive` (opt-in via `streaming.send-keepalive-before-ft`); local helper deleted, local bootstrap test updated to set the opt-in flag |
| `internal/config/config_basic.go` / `handler.go` `persistWith` | upstream inlines the same mutate+save-under-lock pattern |
| `internal/api/server.go` `ETag` CORS exposure | already upstream |
| fork `xai_executor.go` `xaiStatusErr` free-usage/403 remapping | already upstream in `xai_executor_response.go` |
| fork `codex_websockets_executor.go` `*WithProxy` session variants | upstream generalized `websocketSessionTargetChanged` / `detachMismatchedWebsocketSessionConn` to take `proxyURL` |
| fork `sdk/auth/{codex_device,filestore}.go` codex identity enrichment | upstream computes the same plan/hash inline; no consumer for the extra attributes |
| fork `nextRefreshCheckAt` `auth.Disabled` short-circuit | upstream re-allows scheduling normal disabled credentials (issue #6742 recovery) |
| fork `isOpenAIFamily("xai")` | caused xAI level validation to reject clamps that upstream's tests require |

### 5. Deliberate behavior change to review

**`gpt-5.5` is back in the `codex-free` catalog.** The fork removed it in `ec572b91` (2026-04-24); upstream re-added it in `2ea8f77e` (2026-04-29) and `TestCodexConfigurationUpdateCapability` asserts it exists in every tier with `support_configuration_update=false`. The fork's `TestCodexFreeModelsExcludeGPT55` was removed as contradictory.

To restore the exclusion: delete the `gpt-5.5` entry from the `codex-free` array in `internal/registry/models/models.json`. No Go change needed.

### 6. Fork-specific v8 config-safety fixes (new work in this sync)

Upstream v8 comments out unrecognized top-level sections during a configuration write. The fork's three top-level keys were therefore registered as first-class v8 sections so a management write never silently deletes them:

- `v8AllowedRoots()` now includes `xai-proxy-pool`, `xai-resin-proxy`, `commandcode-api-key`.
- `IsV8ConfigLayout()` excludes those three from the "this document is v8" heuristic, so legacy (`/v0`) saves keep the legacy layout.
- `XAIProxyPoolConfig` / `XAIResinProxyConfig` gained `IsZero()` and the fields gained `omitempty`, so an unconfigured fork no longer emits empty blocks into saved config.
- `TestCommandCodeRegistrationAndLocalModels` updated for v8's requirement that an auth be registered in the manager before model registration.

## Verification

```
gofmt -l .                        -> 0
go build ./cmd/server             -> ok
go vet ./...                      -> ok
go test ./...                     -> 103 packages ok, 0 failures
Gate G1 (no /v7 imports)          -> 0
Gate G2 (protected paths)         -> no diff vs cf5070da
Gate G3 (CPA-Manager-Plus refs)   -> 4
conflict markers                  -> 0
```

## Stats

- 34 content conflicts resolved (52 files touched by both sides).
- 1809 staged path changes + 62 follow-up edits.
- 0 protected-path changes.
