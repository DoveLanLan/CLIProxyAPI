# Regression Checklist: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-01
- Related: `spec.md`, `tasks.md`

## Gates (from Repo Snapshot)

- Build: `go build -o test-output ./cmd/server && rm test-output`
- Tests: `go test ./...`
- Format: `gofmt -l` clean on changed Go files
- Whitespace/conflict: `git diff --check`; `git diff --name-only --diff-filter=U` empty; no conflict markers
- Protected paths: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml` empty
- CPA-Manager defaults: `internal/config/config.go` + `internal/managementasset/updater.go` still point at `seakee/CPA-Manager`

## To Execute

### Automated Checks

- [ ] No unresolved merge files: `git diff --name-only --diff-filter=U`
- [ ] No conflict markers: `rg -n '^(<<<<<<<|=======|>>>>>>>)' -S . --glob '!logs/**' --glob '!auths/**' --glob '!tmp/**'`
- [ ] Excluded files unchanged: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml`
- [ ] Formatting: `gofmt -l` on changed Go files
- [ ] Whitespace check: `git diff --check`
- [ ] New upstream subsystems present: `git ls-tree -d --name-only upstream/main` parity for `internal/{pluginhost,pluginstore,signature,safemode,homeplugins,htmlsanitize,httpfetch}` and `sdk/{pluginabi,pluginapi,pluginhost,pluginstore}`
- [ ] CPA-Manager defaults: `git grep -n 'seakee/CPA-Manager' -- internal/config/config.go internal/managementasset/updater.go`
- [ ] Focused Codex invalidated-token fallback: `go test ./sdk/cliproxy/auth -run TestManager_CodexInvalidatedOAuthTokenDisablesAndFallsBackWithMaxRetryOne -v`
- [ ] Focused auth package: `go test ./sdk/cliproxy/auth`
- [ ] Focused OpenAI compat xhigh thinking: `go test ./sdk/cliproxy -run TestServiceOpenAICompatThinking` (and related)
- [ ] Focused OpenAI stream null usage: `go test ./sdk/api/handlers/openai` (null usage chunk test)
- [ ] Focused DeepSeek reasoning echo: `go test ./internal/runtime/executor` / `internal/translator` DeepSeek tests
- [ ] Focused GPT-5.5 Codex free-tier filter: `go test ./internal/registry ./sdk/api/handlers/openai` (codex client models test)
- [ ] Focused string system prompt: `go test ./internal/runtime/executor -run TestCheckSystemInstructionsWithMode` (string system prompt test)
- [ ] New upstream plugin suite: `go test ./internal/pluginhost ./internal/pluginstore ./sdk/pluginhost ./sdk/pluginstore ./internal/signature ./internal/safemode`
- [ ] Full test suite: `go test ./...`
- [ ] Server compile gate: `go build -o test-output ./cmd/server && rm test-output`

### Manual Checks

- [ ] Confirmed `.github` and Docker/deploy-related upstream changes were excluded.
- [ ] Confirmed translator changes are part of the broader upstream sync, not a translator-only task.
- [ ] Confirmed CPA-Manager default repository remains `seakee/CPA-Manager`.
- [ ] Confirmed local OpenAI-compatible `xhigh` thinking defaults and tests remain present after sync.
- [ ] Confirmed local Codex invalidated OAuth token fallback still passes with `max-retry-credentials: 1`.
- [ ] Confirmed DeepSeek model registry helpers and GPT-5.5 free-tier filtering remain.
- [ ] Spot-checked `config.example.yaml` for upstream-added fields (plugin, safemode, `disable-cooling`, `rebuild_mid_system_message`, `max` reasoning) without losing local CPA-Manager/deploy sections.

## Follow-up Regression Areas (pre-deployment)

- Exercise plugin install/delete/hot-reload via management API with real plugin assets.
- Exercise Codex `/v1/responses` WS↔SSE transcript replay, image/video routes, `gpt-image-1.5`, `disable-cooling`, `max` reasoning, `ResetQuota` against production credentials.
- Exercise management auth file upload/delete/list flows in the management UI.
- Confirm production compose/deploy files still build and start (separate deployment task).

## Residual Risk

- Large upstream sync touching runtime routing, translators, SDK handlers, registry data, and management APIs. Automated tests and build pass is required, but production config should still be reviewed before rollout because upstream introduced new config fields, provider behavior, and the plugin subsystem.
