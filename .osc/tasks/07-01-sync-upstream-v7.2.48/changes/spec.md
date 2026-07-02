# Spec: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-01
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## Repo Snapshot

- Modules/components: `cmd/server` binary; server internals under `internal/*`; reusable SDK under `sdk/*`; docs/examples/tests at top level. New upstream subsystems entering scope: `internal/{pluginhost,pluginstore,signature,safemode,homeplugins,htmlsanitize,httpfetch}`, `sdk/{pluginabi,pluginapi,pluginhost,pluginstore}`, `cmd/fetch_codex_models`, `examples/plugin/**`. Confidence: High. Evidence: `git ls-tree -r upstream/main`, `go.mod`, `AGENTS.md`.
- Toolchains: Go 1.26 module `github.com/router-for-me/CLIProxyAPI/v7`; format with `gofmt`; build gate `go build -o test-output ./cmd/server`; tests `go test ./...`. Confidence: High. Evidence: `go.mod` (both branches), `.github/workflows/pr-test-build.yml`, `AGENTS.md`.
- Quality/CI: artifact-first `.osc` workflow; PR build gate compiles `./cmd/server`; `internal/translator/**` protected for ordinary PRs but in scope for this sync; deployment/CI/Docker paths protected by user direction. Confidence: High. Evidence: `.osc/workflow.md`, `.osc/spec/project-spec.md`, prior task `05-22-merge-upstream-non-docker-changes`.

## Scope

### In scope

- Apply upstream changes from `21fad9db..upstream/main` (`v7.2.48`, `956ce7cf`) to the working tree, excluding protected paths.
- Include Go source, tests, registry data, README/docs, examples, assets, module files, and non-protected config templates.
- Include `internal/translator/**` changes as part of the broader upstream sync.
- Re-apply local behavior patches onto upstream's newer structure: Codex invalidated OAuth token failover, OpenAI compat `xhigh` thinking defaults, OpenAI stream null-usage chunk handling, DeepSeek models + reasoning echo normalization, GPT-5.5 Codex support with free-tier filtering, `host.docker.internal` gateway mapping, websocket body-log growth cap, string-form system prompt preservation.
- Preserve CPA-Manager defaults in `internal/config/config.go` and `internal/managementasset/updater.go`.

### Out of scope

- `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.sh`, `docker-build.ps1`, `docker-compose*.yml`, `.env.cluster.example`, `deploy/**`, `.goreleaser.yml`.
- Local `.osc` workflow/task history outside this task's own change artifacts.
- Local `config.yaml`, `auths/`, `logs/`.
- Remote push or production deployment.

## Acceptance Criteria (testable)

1. Upstream non-protected patch is applied; new upstream subsystems (`internal/pluginhost`, `internal/pluginstore`, `internal/signature`, `internal/safemode`, `internal/homeplugins`, `internal/htmlsanitize`, `internal/httpfetch`, `sdk/pluginabi`, `sdk/pluginapi`, `sdk/pluginhost`, `sdk/pluginstore`, `cmd/fetch_codex_models`, `examples/plugin/**`) are present. (Verify: `git ls-tree` / `git diff --name-only` spot checks.)
2. Protected paths match pre-sync `HEAD`. (Verify: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml` is empty.)
3. CPA-Manager defaults preserved. (Verify: `internal/config/config.go` still has `DefaultPanelGitHubRepository = "https://github.com/seakee/CPA-Manager-Plus"`; `internal/managementasset/updater.go` default release/fallback URLs point at `seakee/CPA-Manager-Plus`.)
4. Local behavior patches preserved and green. (Verify: focused `go test` for Codex invalidated-token failover with `max-retry-credentials=1`, OpenAI compat xhigh thinking, OpenAI stream null usage, DeepSeek reasoning echo, GPT-5.5 Codex free-tier filter, string system prompt.)
5. Repo formatted and conflict-free. (Verify: `gofmt -l` clean on changed Go files, `git diff --check`, no `<<<<<<<`/`=======`/`>>>>>>>` markers.)
6. Full test suite passes. (Verify: `go test ./...`.)
7. Server compile gate passes. (Verify: `go build -o test-output ./cmd/server && rm test-output`.)

## Behavior / Requirements

- Conflict priority: protected path → local `HEAD`; CPA-Manager defaults → local; local behavior patch → local behavior adapted to upstream structure; everything else → upstream.
- The final tree contains upstream functionality for the plugin system, signature validation, safemode, home plugins, image/video handling, `gpt-image-1.5`, `disable-cooling`, `max` reasoning depth, `ResetQuota`, Codex WS↔SSE transcript replay, and translator/runtime/registry/auth/management/SDK updates through `v7.2.48`.
- Docker, GitHub workflow, and deployment changes from upstream remain absent.
- No secrets logged; upstream logging changes must preserve masking behavior.

## Edge Cases

- If an upstream hunk conflicts with local code, resolve manually: preserve local behavior, adapt to upstream's new call sites/structure, and run the corresponding focused test.
- If excluding Docker/deploy files leaves a new non-Docker reference to a missing Docker artifact, keep the reference only if documentation-only and non-breaking.
- If upstream renamed or removed a symbol the fork's local patches reference, re-port the patch to the new symbol and add/adjust a focused test.
- If generated/large registry JSON changes conflict, prefer valid upstream data and run registry-parsing tests.
- If upstream's plugin system changes management asset handling in a way that touches CPA-Manager defaults, re-assert CPA-Manager defaults after taking upstream's structural changes.

## Compatibility Notes

- Backwards compatibility: existing local config and auth files must continue to load; upstream-added config fields must keep old configs valid.
- Data/migrations: no database schema migration expected.
- Config/flags: upstream may add config fields (e.g. plugin, safemode, `disable-cooling`, `rebuild_mid_system_message`); defaults must keep old configs valid. CPA-Manager default panel repo/URL unchanged.

## API/UX Decisions

- Inputs/outputs: upstream API route additions (management plugin endpoints, `ResetQuota`, image/video, health) are accepted; public `/v1`, `/v1beta` compatibility follows upstream.
- States/errors: keep upstream error handling unless it conflicts with local user-facing behavior (Codex failover messaging, xhigh thinking defaults).
- Telemetry/logging: upstream logging changes accepted outside protected files; secret masking preserved.

---

## Addendum: Remove Legacy CPA-Manager from Production Deploy

- Date: 2026-07-02
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

### Repo Snapshot

- Modules/components: production deployment assets live under `deploy/`; GitHub production automation lives under `.github/workflows/`; runtime server remains `cmd/server` and is not changed by this hotfix. Confidence: High. Evidence: `deploy/compose.production.yml`, `deploy/scripts/remote-deploy.sh`, `.github/workflows/deploy-production.yml`.
- Toolchains: production deploy packages `deploy/` and runs `scripts/remote-deploy.sh`; the script executes `docker compose pull`, `docker compose up -d --remove-orphans`, and gateway nginx reload. Confidence: High. Evidence: `.github/workflows/deploy-production.yml`, `deploy/scripts/remote-deploy.sh`.
- Quality/CI: no Go code changes expected; validation should include compose config rendering and the standard server compile gate if any Go files are touched. Confidence: High. Evidence: `AGENTS.md`, `.github/workflows/pr-test-build.yml`.

### Scope

#### In scope

- `deploy/compose.production.yml`: remove the legacy `cpa-manager` service.
- `deploy/.env.example`: remove obsolete production env vars used only by the removed service.
- `deploy/README.md`: update topology, bootstrap, and management access docs for CPA Manager Plus via the CLIProxyAPI management panel.
- `.github/workflows/update-cpa-manager-image.yml`: remove obsolete automation for the retired service.
- `.osc/spec/project-spec.md` and task change artifacts: record the hotfix.

#### Out of scope

- Go runtime/source behavior.
- `docker-compose.yml` and `docker-compose.example.yml` for local development.
- Remote VPS state cleanup beyond compose orphan removal.
- CPA Manager Plus external deployment mechanics, if any.

### Acceptance Criteria (testable)

1. Production compose no longer defines `cpa-manager` and no longer binds `${TAILSCALE_CPA_MANAGER_PORT:-18318}`. (Verify: `rg -n "cpa-manager|TAILSCALE_CPA_MANAGER_PORT|18318" deploy/compose.production.yml deploy/.env.example` returns no matches.)
2. Production deploy still defines `cli-proxy-api` and the external `${GATEWAY_NETWORK:-vps-gateway}` network. (Verify: `docker compose -f deploy/compose.production.yml --env-file deploy/.env.example config --services` includes `cli-proxy-api` only for the base file.)
3. Obsolete CPA-Manager image update workflow is removed. (Verify: `.github/workflows/update-cpa-manager-image.yml` is absent.)
4. Deployment docs no longer instruct operators to expose the legacy CPA-Manager Usage Service. (Verify: `rg -n "Usage Service|TAILSCALE_CPA_MANAGER_PORT|CPA_MANAGER_IMAGE|CPA_MANAGER_USAGE_QUERY_LIMIT|18318" deploy/README.md` returns no matches.)
5. Standard Go compile gate remains green if Go files are touched. (Verify: `go build -o test-output ./cmd/server && rm test-output`; skip if only docs/deploy files change.)

### Behavior / Requirements

- The base production stack manages only `cli-proxy-api`; optional split-proxy remains controlled by `ENABLE_SPLIT_PROXY=true`.
- CPA Manager Plus is reached through the configured CLIProxyAPI management panel on the private management port.
- The deploy script's existing `--remove-orphans` behavior should remove the previous compose-managed `cpa-manager` container after the updated compose file reaches the server.
- No host port `18318` is reserved by this production stack after the change.

### Edge Cases

- If a manually created external container already uses `18318`, deploy should still succeed because this stack no longer binds that port.
- If a stale compose-managed `cpa-manager` container exists from older deploys, `--remove-orphans` should remove it when the compose project name is unchanged.
- If an operator still needs a standalone usage collector, it must be managed outside this repository and must not reuse this stack's service name.

### Compatibility Notes

- Backwards compatibility: existing CLIProxyAPI config and auth volume paths stay unchanged.
- Data/migrations: no data migration; old `data/cpa-manager/` contents are left untouched on the server.
- Config/flags: `CPA_MANAGER_*` and `TAILSCALE_CPA_MANAGER_PORT` production `.env` entries become unused and can be manually removed from the VPS `.env`.

### API/UX Decisions

- Inputs/outputs: no API changes.
- States/errors: production deploy should no longer fail on `Bind for 100.67.99.9:18318 failed: port is already allocated`.
- Telemetry/logging: no logging changes.
