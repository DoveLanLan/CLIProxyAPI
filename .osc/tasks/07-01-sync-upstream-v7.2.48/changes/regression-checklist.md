# Regression Checklist: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-01
- Related: `spec.md`, `tasks.md`

## Gates (from Repo Snapshot)

- Build: `go build -o test-output ./cmd/server && rm test-output`
- Tests: `go test ./...`
- Format: `gofmt` on changed Go files
- Whitespace/conflict: `git diff --cached --check`; `git diff --name-only --diff-filter=U` empty; no conflict markers
- Protected paths: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml` only shows intentional `deploy/README.md` Plus documentation update
- CPA-Manager defaults: `internal/config/config.go` + `internal/managementasset/updater.go` point at `seakee/CPA-Manager-Plus`

## Executed

### Automated Checks

- [x] No unresolved merge files: `git diff --name-only --diff-filter=U` returned `0`.
- [x] No conflict markers: grep check over Go/YAML/Markdown returned no markers outside ignored artifacts.
- [x] Excluded files unchanged: protected-path check showed only intentional `deploy/README.md` doc update.
- [x] Formatting: `gofmt` on changed Go files.
- [x] Whitespace check: `git diff --cached --check` passed.
- [x] New upstream subsystems present: pluginhost/pluginstore/signature/safemode/homeplugins/htmlsanitize/httpfetch and sdk plugin packages are present.
- [x] CPA-Manager-Plus defaults: `git grep -n 'seakee/CPA-Manager-Plus' -- internal/config/config.go internal/managementasset/updater.go config.example.yaml README.md README_CN.md README_JA.md deploy/README.md`.
- [x] Focused Codex invalidated-token fallback: `go test ./sdk/cliproxy/auth -run 'TestManager_CodexInvalidatedOAuthTokenDisablesAndFallsBackWithMaxRetryOne|TestManager_CodexGeneric401UsesTemporaryCooldownAndMaxRetryLimit' -v`.
- [x] Focused auth package: `go test ./sdk/cliproxy/auth -run 'TestManager_Codex|TestManagerOverrides|TestFile|TestSelector' -v`.
- [x] Focused OpenAI compat xhigh thinking: `go test ./sdk/cliproxy -run TestOpenAICompat -v`.
- [x] Focused OpenAI stream null usage: `go test ./internal/runtime/executor/helps -run 'TestParseOpenAIStreamUsage' -v`.
- [x] Focused GPT-5.5 Codex free-tier filter and registry: `go test ./internal/registry -run 'TestCodex|TestValidate|TestStatic|TestGet|TestAntigravity|TestWithXAI' -v`.
- [x] Focused string system prompt: `go test ./internal/runtime/executor -run 'Test.*System.*String|TestCheckSystemInstructionsWithMode' -v`.
- [x] Focused websocket log cap: `go test ./sdk/api/handlers/openai -run 'TestAppendWebsocketEvent' -v`.
- [x] Focused management/CPA usage: `go test ./internal/api/handlers/management -run 'Test.*Usage|Test.*AuthFiles|Test.*Config|Test.*Handler' -v`.
- [x] Focused redisqueue: `go test ./internal/redisqueue -v`.
- [x] Focused OpenAI image/video/websocket handlers: `go test ./sdk/api/handlers/openai -run 'Test.*Usage|Test.*Codex|TestAppendWebsocketEvent|Test.*Images|Test.*Videos' -v`.
- [x] Full test suite: `go test ./...` passed.
- [x] Server compile gate: `go build -o test-output ./cmd/server && rm test-output` passed.

### Manual Checks

- [x] Confirmed `.github` and Docker/deploy-related upstream changes were excluded except the intentional `deploy/README.md` Plus doc update.
- [x] Confirmed translator changes are part of the broader upstream sync, not a translator-only task.
- [x] Confirmed CPA-Manager default repository remains `seakee/CPA-Manager-Plus`.
- [x] Confirmed local OpenAI-compatible `xhigh` thinking defaults and tests remain present after sync.
- [x] Confirmed local Codex invalidated OAuth token fallback still passes with `max-retry-credentials: 1`.
- [x] Confirmed DeepSeek model registry helpers and GPT-5.5 free-tier filtering remain.
- [x] Confirmed upstream-deleted Amp and Gemini CLI paths are removed.
- [x] Spot-checked `config.example.yaml` for upstream-added fields (plugin, safemode, `disable-cooling`, image/video, cooldown) without losing CPA-Manager-Plus defaults.

## Follow-up Regression Areas (pre-deployment)

- Exercise plugin install/delete/hot-reload via management API with real plugin assets.
- Exercise Codex `/v1/responses` WS↔SSE transcript replay, image/video routes, `gpt-image-1.5`, `disable-cooling`, `max` reasoning, `ResetQuota` against production credentials.
- Exercise management auth file upload/delete/list flows in the management UI.
- Confirm production compose/deploy files still build and start (separate deployment task).

## Residual Risk

- Large upstream sync touching runtime routing, translators, SDK handlers, registry data, and management APIs. Automated tests and build pass, but production config should still be reviewed before rollout because upstream introduced new config fields, provider behavior, and the plugin subsystem.
