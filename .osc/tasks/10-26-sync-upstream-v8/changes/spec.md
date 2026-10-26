# Spec: Sync Upstream v8.0.43

- Date: 2026-10-26
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## 1. Sync identity

| Item | Value |
|---|---|
| Pre-sync local HEAD (rollback point) | `cf5070da` (`v7.2.93-94-gcf5070da`) |
| Last sync point | `e92bd900` (merged upstream `01f387f4` = `v7.2.93`, 2026-07-21) |
| Merge-base (`HEAD` ∩ `upstream/main`) | `01f387f4` |
| Target upstream | `upstream/main` @ `v8.0.43` |
| Upstream commits to absorb | `1441` |
| Upstream files changed since merge-base | `1810` |
| Local files changed since merge-base | `552` |
| Overlapping files | `52` |
| Expected content conflicts | `34` |

## 2. Version / module contract

- `module github.com/router-for-me/CLIProxyAPI/v8`
- `go 1.26.10`
- New direct deps carried in: `pion/webrtc/v4`, `pion/ice/v4`, `pion/interceptor`, `pion/rtp`, `pion/sdp/v3`, `pion/stun/v3`, `libp2p/zeroconf/v2`, `go-git v6.0.0-alpha.5`, `tiktoken-go/tokenizer v0.8.1`.
- **Gate G1**: `grep -rl 'CLIProxyAPI/v7' --include='*.go' .` returns **0 lines**.

## 3. Protected paths (keep local, byte-identical to `cf5070da`)

```
.github/**
Dockerfile
.dockerignore
docker-build.sh
docker-build.ps1
docker-compose*.yml
.env.cluster.example
deploy/**
.goreleaser.yml
.osc/**
.claude/**
```

**Gate G2**: `git diff --stat cf5070da -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml` shows no changes.

## 4. CPA-Manager contract

- `internal/config/config.go`: `DefaultPanelGitHubRepository = "https://github.com/seakee/CPA-Manager-Plus"`.
- `internal/managementasset/updater.go`: release API + fallback download URLs point to `seakee/CPA-Manager-Plus`.
- `config.example.yaml`: panel repo references stay local.

**Gate G3**: `git grep -n 'seakee/CPA-Manager-Plus' -- internal/config/config.go internal/managementasset/updater.go config.example.yaml` returns >= 2 hits and no `router-for-me/CPA-Manager` default is reintroduced.

## 5. Local provider/protocol contracts (must remain functional)

| Area | Key files | Contract |
|---|---|---|
| xAI rotating proxy pool | `internal/runtime/executor/{xai_proxy_pool_executor,xai_executor}.go`, `helps/xai_proxy_pool.go`, `internal/api/handlers/management/xai_proxy_pool.go`, `internal/config/xai_proxy_pool*` | management API + executor binding + config parse survive upstream refactors |
| xAI Resin | `helps/xai_resin_proxy*.go`, `deploy/XAI_RESIN_PROXY_SETUP_CN.md` | lease rotation on exact 402, retry pre-response failures |
| Command Code provider | `commandcode_executor.go`, `helps/commandcode_*.go`, `sdk/cliproxy/{commandcode_models.go,auth/commandcode.go}` | native provider registered + stream works |
| DeepSeek | `helps/openai_compat_deepseek.go`, `internal/registry/*` overrides | model list + reasoning echo + effort levels |
| Vision preflight | `helps/openai_tool_images.go`, `helps/openai_message_links.go` | tool-result images moved into user messages; DeepSeek vision allowed |
| Thinking | `internal/thinking/validate.go`, `provider/xai/apply.go`, `sdk/cliproxy/config_model_thinking_test.go` | `xhigh` defaults; per-model override for `claude-api-key` |
| Codex | `internal/runtime/executor/codex_websockets_executor.go`, `sdk/auth/codex_device.go` | invalidated-token failover; GPT-5.5 free-tier filter; body-log growth cap |
| Claude | `sdk/api/handlers/claude/code_handlers.go`, `sdk/api/handlers/handlers.go` | upstream error metadata retained; string system prompt preserved |

## 6. Superseded local patches (adopt upstream)

| Local commit | Upstream replacement | Action |
|---|---|---|
| `9e5bf4d5 fix(antigravity): accept array tool schemas without items` | `b5cde4ba fix(gemini): add default items schema for tool array definitions` | take upstream, drop local patch |

## 7. Conflict resolution rules

1. Protected path → keep local `HEAD` (`git checkout cf5070da -- <path>`).
2. CPA-Manager default → keep local value inside otherwise-upstream file.
3. Local provider patch → keep local behavior, port to v8 structure; if upstream restructured the call site, follow upstream shape and re-inject behavior.
4. Everything else → take upstream (`git checkout --theirs` equivalent / accept upstream side).
5. No conflict markers may remain: `git diff --check` + `grep -rn '^<<<<<<<' --include='*.go'`.

## 8. Definition of Done

- [ ] `task/sync-upstream-v8` branched from `cf5070da`, merged `upstream/main`.
- [ ] 0 unresolved paths (`git diff --name-only --diff-filter=U`).
- [ ] Gate G1 (no `/v7` imports), G2 (protected paths untouched), G3 (CPA-Manager) pass.
- [ ] `gofmt -l .` clean for changed Go files.
- [ ] `go build -o test-output ./cmd/server` succeeds, then remove `test-output`.
- [ ] `go vet ./...` passes.
- [ ] `go test ./...` passes (or documented, pre-existing-only failures).
- [ ] Focused provider tests pass: xai pool, commandcode, deepseek, thinking, antigravity schema.
- [ ] Closure artifacts written: `change-summary.md`, `regression-checklist.md`, `rollback-notes.md`, `.osc/quality-gate.md`.
