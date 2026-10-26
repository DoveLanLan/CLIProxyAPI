# Regression Checklist: Sync Upstream v8.0.43

- Date: 2026-10-26

## Automated gates

| Gate | Command | Expected | Result |
|---|---|---|---|
| Formatting | `gofmt -l .` | empty | pass |
| Build | `go build -o test-output ./cmd/server && rm test-output` | exit 0 | pass |
| Vet (compiles tests) | `go vet ./...` | exit 0 | pass |
| Full tests | `go test ./...` | all ok | 103 ok / 0 fail |
| G1 module path | `grep -rl 'CLIProxyAPI/v7' --include='*.go' .` | 0 | pass |
| G2 protected paths | `git diff --stat cf5070da -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml` | empty | pass |
| G3 CPA-Manager | `git grep -c 'seakee/CPA-Manager-Plus' -- internal/config/config_defaults.go internal/managementasset/updater.go config.example.yaml` | 1/1/2 | pass |
| Conflicts | `git diff --name-only --diff-filter=U` + marker grep | 0 | pass |

## Focused suites (fork-critical)

| Area | Test(s) | Result |
|---|---|---|
| Antigravity schema/translator | `go test ./internal/translator/antigravity/... ./internal/runtime/executor/ -run TestAntigravity` | pass |
| Antigravity signature/CAQS | `go test ./internal/signature/... ./internal/runtime/executor/ -run 'TestAntigravityCAQS|TestClaudeAntigravity'` | pass |
| xAI proxy pool + Resin | `go test ./internal/runtime/executor/ -run 'TestXAIProxyPool|TestXAIResin'` | pass |
| xAI thinking | `go test ./internal/runtime/executor/ -run TestXAIExecutorThinkingPayloadOverride` + `go test ./test/ -run TestThinkingE2EProviderTargets` | pass |
| xAI status/stream errors | `go test ./internal/runtime/executor/ -run TestXAIStatusErr` / `TestXAIStreamEventError` | pass |
| Command Code | `go test ./sdk/cliproxy/ -run TestCommandCode` | pass |
| DeepSeek reasoning echo | `go test ./internal/runtime/executor/ -run TestOpenAICompatExecutorDeepSeek` | pass |
| Tool images / message links | `go test ./internal/runtime/executor/ -run 'TestOpenAICompatExecutor'` | pass |
| OpenAI-compat thinking | `go test ./sdk/cliproxy/ -run TestOpenAICompatExplicitThinkingRemainsAuthoritative` | pass |
| Claude error metadata | `go test ./sdk/api/handlers/claude/` | pass |
| Pre-first-token keepalive | `go test ./sdk/api/handlers/claude/ -run TestClaudeStreamingBootstrapHeartbeatPrecedesDelayedStartupError` | pass |
| Codex rejected token (upstream #6742) | `go test ./sdk/cliproxy/auth/ -run TestIssue6742` | pass |
| Config v8 migration/layout | `go test ./internal/config/` | pass |
| Legacy v0 save layout | `go test ./internal/api/handlers/management/ -run TestV0RepeatedSavesKeepLegacyClientAlias` | pass |
| Config diff redaction | `go test ./internal/watcher/diff/` | pass |

## Manual / post-deploy smoke checks (production)

These cannot run in CI and must be verified after the auto-deploy completes:

1. **Startup** — container boots, `/v1/models` responds, management panel reachable.
2. **Config not clobbered** — after first `/v0/management` write, `xai-proxy-pool`, `xai-resin-proxy`, `commandcode-api-key` still present in `config.yaml` and not commented out.
3. **CPA-Manager panel** — still served from `seakee/CPA-Manager-Plus`; version banner loads.
4. **xAI routing** — one request through the Resin/pool path succeeds; `/v0/management/xai-proxy-pool/status` returns 200.
5. **Command Code** — one `cc/<alias>` request succeeds.
6. **DeepSeek** — tool-followup turn in thinking mode succeeds (no `reasoning_content` 400).
7. **Antigravity** — one Claude-model and one Gemini-model request succeed; tool-call round trip works.
8. **Codex free tier** — `gpt-5.5` now appears in the free tier model list (intended change; see change summary §5).

## Known follow-ups

- `xai-proxy-pool` / `xai-resin-proxy` / `commandcode-api-key` are fork-only sections. They are registered in `v8AllowedRoots()` but have no `/v8/management` route mapping; the panel keeps using `/v0/management/xai-proxy-pool/**`.
- Upstream's v8 layout has no `commandcode` group under `api-keys`; the fork keeps the legacy root list form.
