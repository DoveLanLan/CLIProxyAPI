# Quality Gate: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-02
- Scope: Upstream `router-for-me/CLIProxyAPI` sync to `v7.2.48` (`956ce7cf`) while preserving local deploy/ops and CPA-Manager-Plus defaults.

## Assumptions

- The sync stays local on `task/sync-upstream-v7.2.48`; push and production deployment are separate.
- Protected deployment paths are preserved except intentional `deploy/README.md` panel-source documentation update.
- Upstream removal of Amp and Gemini CLI code paths is accepted.

## Detected Gates

- Gate Name: Server compile
  - Confidence: High
  - Evidence: `AGENTS.md`, PR build convention
- Gate Name: Full Go tests
  - Confidence: High
  - Evidence: `AGENTS.md`, broad protocol/runtime/auth/management changes
- Gate Name: Conflict/format check
  - Confidence: High
  - Evidence: large merge with prior conflicts
- Gate Name: Local behavior regression checks
  - Confidence: High
  - Evidence: fork-specific patches re-ported during sync

## Executed Gates

1. Conflict check
   - Command: `git diff --name-only --diff-filter=U`
   - Result: Passed; `0` unresolved files.
2. Conflict marker scan
   - Command: grep scan over Go/YAML/Markdown files for `<<<<<<<`, `=======`, `>>>>>>>`
   - Result: Passed; no markers found outside ignored artifacts.
3. Formatting
   - Command: `gofmt` on changed Go files
   - Result: Passed.
4. Whitespace check
   - Command: `git diff --cached --check`
   - Result: Passed.
5. Server build
   - Command: `go build -o test-output ./cmd/server && rm test-output`
   - Result: Passed.
6. Full tests
   - Command: `go test ./...`
   - Result: Passed.
7. Focused local behavior checks
   - Commands included:
     - `go test ./sdk/cliproxy/auth -run 'TestManager_CodexInvalidatedOAuthTokenDisablesAndFallsBackWithMaxRetryOne|TestManager_CodexGeneric401UsesTemporaryCooldownAndMaxRetryLimit' -v`
     - `go test ./sdk/cliproxy -run TestOpenAICompat -v`
     - `go test ./internal/runtime/executor/helps -run 'TestParseOpenAIStreamUsage' -v`
     - `go test ./internal/runtime/executor -run 'Test.*System.*String|TestCheckSystemInstructionsWithMode' -v`
     - `go test ./sdk/api/handlers/openai -run 'TestAppendWebsocketEvent' -v`
     - `go test ./internal/registry -run 'TestCodex|TestValidate|TestStatic|TestGet|TestAntigravity|TestWithXAI' -v`
     - `go test ./internal/api/handlers/management -run 'Test.*Usage|Test.*AuthFiles|Test.*Config|Test.*Handler' -v`
     - `go test ./internal/redisqueue -v`
   - Result: Passed.
8. CPA-Manager-Plus default check
   - Command: `git grep -n 'seakee/CPA-Manager-Plus' -- internal/config/config.go internal/managementasset/updater.go config.example.yaml README.md README_CN.md README_JA.md deploy/README.md`
   - Result: Passed.

## Final Self-Review

- Security & secrets: No secrets added; `config.yaml` remains ignored; management panel source points to public `seakee/CPA-Manager-Plus` release assets.
- Edge cases & error handling: Codex invalidated OAuth token handling disables bad auth and persists disabled reason; generic 401 remains cooldown-limited.
- Backward compatibility / migrations: No database/storage migration introduced; upstream config additions should keep old configs valid.
- API/contract compatibility: Upstream API additions accepted; local protected deployment paths preserved.
- Observability: WebSocket body log growth remains capped; upstream logging/timeline changes retained.
- Config/env changes: `config.example.yaml` now documents upstream plugin/safemode/image/video/cooldown fields and local CPA-Manager-Plus panel source.
- Rollback plan: revert sync commit/series or reset to `307f5ff8` if not pushed; see rollback notes.

## PR-ready checklist

- [x] Change-workflow artifacts updated
- [x] No unresolved conflicts
- [x] `git diff --cached --check`
- [x] `go build -o test-output ./cmd/server && rm test-output`
- [x] `go test ./...`
- [x] Focused local behavior tests
- [x] CPA-Manager-Plus defaults verified

---

# Quality Gate Addendum: Remove Legacy CPA-Manager from Production Deploy

- Date: 2026-07-02
- Scope: Production deploy hotfix for the old standalone CPA-Manager service and port conflict on `100.67.99.9:18318`.

## Assumptions

- Production now uses CPA Manager Plus through the CLIProxyAPI management panel.
- No Go runtime behavior changed in this hotfix.
- The existing deploy script keeps using `docker compose up -d --remove-orphans`.

## Suspected Change Scope

- `deploy/compose.production.yml`
- `deploy/.env.example`
- `deploy/README.md`
- `.github/workflows/update-cpa-manager-image.yml`
- `.osc/spec/project-spec.md`
- `.osc/tasks/07-01-sync-upstream-v7.2.48/changes/*`

## Detected Gates

- Gate Name: Production compose render
  - Confidence: High
  - Evidence: `.github/workflows/deploy-production.yml` step `Deploy production stack`; `deploy/scripts/remote-deploy.sh` runs `docker compose pull` and `docker compose up -d --remove-orphans`.
- Gate Name: Split-proxy compose render
  - Confidence: High
  - Evidence: `deploy/scripts/remote-deploy.sh` appends `deploy/compose.production.split-proxy.yml` when `ENABLE_SPLIT_PROXY=true`.
- Gate Name: Server compile
  - Confidence: High
  - Evidence: `AGENTS.md`; `.github/workflows/pr-test-build.yml` build step.
- Gate Name: Stale reference scan
  - Confidence: Medium
  - Evidence: deployment failure log and removed service/environment keys.

## Suggested Gate Run (Local)

1. `docker compose -f deploy/compose.production.yml --env-file deploy/.env.example config --services`
   - Rationale: verify the base production stack no longer includes the old service.
2. `docker compose -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml --env-file deploy/.env.example config --services`
   - Rationale: verify optional split-proxy still composes with the base stack.
3. `rg -n "Usage Service|TAILSCALE_CPA_MANAGER_PORT|18318|CPA_MANAGER_IMAGE|CPA_MANAGER_USAGE_QUERY_LIMIT|cpa-manager" deploy .github/workflows`
   - Rationale: verify production deploy docs/workflows no longer carry legacy service references.
4. `go build -o test-output ./cmd/server && rm test-output`
   - Rationale: preserve the repo's standard compile gate.

## Executed Gates

1. Production compose render
   - Command: `docker compose -f deploy/compose.production.yml --env-file deploy/.env.example config --services`
   - Result: Passed; output was `cli-proxy-api`.
2. Split-proxy compose render
   - Command: `docker compose -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml --env-file deploy/.env.example config --services`
   - Result: Passed; output was `split-proxy` and `cli-proxy-api`.
3. Stale reference scan
   - Command: `rg -n "Usage Service|TAILSCALE_CPA_MANAGER_PORT|18318|CPA_MANAGER_IMAGE|CPA_MANAGER_USAGE_QUERY_LIMIT|cpa-manager" deploy .github/workflows`
   - Result: Passed; no stale legacy production refs.
4. Workflow removal check
   - Command: `test ! -f .github/workflows/update-cpa-manager-image.yml`
   - Result: Passed.
5. Server compile
   - Command: `go build -o test-output ./cmd/server && rm test-output`
   - Result: Passed.

## Final Self-Review

- Security & secrets: No secrets added; obsolete `.env.example` secret placeholders for the old service were removed.
- Edge cases & error handling: External owners of the old port no longer block this stack because it no longer binds that port.
- Backward compatibility / migrations: No data migration; old remote data directories are untouched.
- API/contract compatibility: No public API changes.
- Observability: No logging or metrics changes.
- Config/env changes: Removed unused production-only env sample keys; docs updated.
- Performance risk: No runtime hot-path changes.
- Rollback plan: Revert the hotfix files, but only after ensuring the old port is free.

## PR-ready checklist

- [x] `docker compose -f deploy/compose.production.yml --env-file deploy/.env.example config --services`
- [x] `docker compose -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml --env-file deploy/.env.example config --services`
- [x] Stale legacy production reference scan
- [x] `.github/workflows/update-cpa-manager-image.yml` absent
- [x] `go build -o test-output ./cmd/server && rm test-output`
