# CLIProxyAPI Project Spec

## What This Repository Is For

CLIProxyAPI is a Go-based gateway for AI coding clients and SDKs. It lets tools that speak OpenAI-, Gemini-, Claude-, or Codex-style protocols use a single local or hosted proxy endpoint while the proxy handles:

- provider-specific auth flows such as OAuth/device login
- multi-account routing and retry/failover
- model aliasing, exclusion, and provider remapping
- management APIs for config/auth/logging/usage control
- optional terminal management UI
- embeddable SDK usage through `sdk/cliproxy`

This repository is not a conventional browser frontend project. The interactive surfaces in-tree are:

- the HTTP server in `cmd/server`
- CLI login flows in `internal/cmd`
- the Bubble Tea terminal UI in `internal/tui`
- an updater/serving path for an externally published management panel asset in `internal/managementasset`

## Current OSC Context

- Repo root: `/root/Projects/Go/src/CLIProxyAPI`
- Developer: `hewei`
- Current task: `.osc/tasks/07-27-extract-egress-proxy-pool`
- Immediate implication: Mihomo control-plane ownership is moving to the sibling `EgressProxyPool` project while xAI response interpretation remains in this repository.

**Repo Snapshot**
- **Modules/Components:** `cmd/server` is the runnable server entrypoint; `internal/{api,auth,cmd,config,logging,managementasset,runtime,store,tui,usage,watcher,wsrelay,...}` contains server-only runtime code; `sdk/{api,auth,cliproxy,config,logging,translator}` is the reusable embedding surface; `docs/`, `examples/`, and `test/` hold consumer docs, samples, and regression coverage. (confidence: High) — evidence: `README.md`, `cmd/server/main.go`, `internal/`, `sdk/`, `docs/`, `examples/`, `test/`
- **Toolchains:** build uses Go modules plus `go build ./cmd/server`, a multi-stage Docker image, and GoReleaser archives (High); tests exist both as package tests and top-level regressions under `test/`, but the shallow CI inventory only enforces build on pull requests (Medium); no dedicated lint/staticcheck config was found at depth <= 2, so explicit source-format enforcement is mostly standard Go tooling plus review convention (Low). — evidence: `go.mod`, `Dockerfile`, `docker-compose.yml`, `.goreleaser.yml`, `.github/workflows/pr-test-build.yml`, `.github/workflows/release.yaml`, `.github/workflows/docker-image.yml`
- **Style/Format Enforcement:** the strongest enforced process rules are the `osc` artifact-first workflow, protected `internal/translator/**` PR boundary, and comment-preserving config persistence through management handlers. (confidence: Medium) — evidence: `.osc/workflow.md`, `AGENTS.md`, `CLAUDE.md`, `.github/workflows/pr-path-guard.yml`, `internal/api/handlers/management/config_basic.go`, `internal/api/handlers/management/handler.go`
- **CI Gates/Expectations:** pull requests must compile `./cmd/server`; pull requests touching `internal/translator/**` are rejected; main/tag pushes publish Docker images, and successful main Docker builds trigger the production deploy workflow. Tag pushes publish GoReleaser artifacts with embedded version metadata. (confidence: High) — evidence: `.github/workflows/pr-test-build.yml`, `.github/workflows/pr-path-guard.yml`, `.github/workflows/docker-image.yml`, `.github/workflows/deploy-production.yml`, `.github/workflows/release.yaml`, `.goreleaser.yml`, `Dockerfile`, `deploy/scripts/remote-deploy.sh`
- **Open Questions (max 1):** None. The extraction task is selected and contains its proposal, spec, and task artifacts.

## Product Surfaces

### 1. Proxy Server

The main binary starts an HTTP server that exposes:

- OpenAI-compatible routes under `/v1`
- Gemini-compatible routes under `/v1beta`
- management routes under `/v0/management` when a management secret is configured
- OAuth callback routes on the main server
- optional WebSocket relay support under `/v1/ws`

Primary evidence: `cmd/server/main.go`, `internal/api/server.go`

### 2. Embeddable SDK

`sdk/cliproxy` exposes the same proxy lifecycle as a library so external Go programs can embed config loading, auth refresh, watchers, routes, and usage hooks without shelling out to the CLI binary.

Primary evidence: `docs/sdk-usage.md`, `docs/sdk-advanced.md`, `docs/sdk-watcher.md`, `sdk/cliproxy/builder.go`, `sdk/cliproxy/service.go`

### 3. Management and Operations

Operational control flows through:

- YAML config and auth files
- the Management API for runtime toggles and persistence
- optional file, PostgreSQL, Git, or S3-compatible backing stores
- the Bubble Tea TUI for interactive local management
- the downloaded `management.html` control panel asset

Primary evidence: `config.example.yaml`, `internal/api/handlers/management/`, `internal/store/`, `internal/tui/`, `internal/managementasset/updater.go`

## Architecture Map

| Area | Responsibility | Key paths |
|---|---|---|
| Entry and process boot | flags, environment bootstrapping, store selection, config loading, TUI/server startup | `cmd/server/main.go` |
| HTTP server | Gin engine, middleware, route registration, management route gating | `internal/api/server.go`, `internal/api/middleware/`, `internal/api/modules/` |
| Public protocol handling | OpenAI/Gemini/Claude/Codex-compatible request handling and error payloads | `sdk/api/handlers/` |
| Provider auth and runtime execution | auth manager, selectors, schedulers, execution context, retries | `sdk/cliproxy/auth/`, `sdk/cliproxy/executor/`, `internal/runtime/` |
| Config and persistence | YAML config parsing plus file/Postgres/Git/Object-backed state mirrors | `internal/config/`, `internal/store/` |
| Logging and usage | global log formatter, request logging, request IDs, usage plugins | `internal/logging/`, `internal/usage/`, `sdk/cliproxy/usage/` |
| UI surfaces | terminal UI and external management panel asset handling | `internal/tui/`, `internal/managementasset/` |

#### A) Architecture & boundaries
1. Keep the runnable server entrypoint in `cmd/server` and preserve linker-injected build metadata (`main.Version`, `main.Commit`, `main.BuildDate`) across local builds, Docker builds, and release packaging. — Evidence: `cmd/server/main.go`, `Dockerfile`, `.goreleaser.yml`, `.github/workflows/release.yaml`, `.github/workflows/docker-image.yml` (Documented; confidence: High)
2. Treat `internal/*` as repository-private runtime implementation and `sdk/*` as the supported embedding and protocol surface. If a change is intended for external integrators, it should normally land in `sdk/*` or be documented as server-only. — Evidence: `README.md`, `docs/sdk-usage.md`, `docs/sdk-advanced.md`, `internal/`, `sdk/` (Inferred; confidence: High)
3. Use `internal/api/modules` for optional server route bundles instead of hard-wiring every extension into core route setup. New modules should implement the V2 registration context when possible. — Evidence: `internal/api/modules/modules.go`, `internal/api/server.go` (Documented; confidence: High)
4. Do not modify `internal/translator/**` as ordinary pull-request work; that path is guarded by CI and requires escalation through the repository’s maintenance process. — Evidence: `.github/workflows/pr-path-guard.yml` (Documented; confidence: High)
5. Preserve the separation between protocol-compatible public APIs and management APIs. `/v1` and `/v1beta` must stay client-compatible; `/v0/management` is for local/operator control. — Evidence: `internal/api/server.go`, `sdk/api/handlers/handlers.go`, `internal/api/handlers/management/` (Documented; confidence: High)

#### B) Directory layout & naming
1. Put runnable binaries under `cmd/`, server-only internals under `internal/<domain>`, and reusable library code under `sdk/<domain>`. — Evidence: `cmd/server`, `internal/`, `sdk/` (Inferred; confidence: High)
2. Keep consumer-facing documentation in `docs/`, concrete runnable examples in `examples/`, and cross-package regression tests in top-level `test/`. — Evidence: `docs/`, `examples/`, `test/`, `README.md` (Documented; confidence: High)
3. Follow the existing lowercase package naming and descriptive file suffix patterns such as `*_test.go`, `*_handlers.go`, `*_login.go`, and `*_store.go`. — Evidence: `internal/cmd/`, `internal/api/handlers/management/`, `sdk/api/handlers/`, `internal/store/` (Inferred; confidence: Medium)

#### C) Code style & patterns
1. Target Go `1.26` and keep the module path `github.com/router-for-me/CLIProxyAPI/v7` intact when changing imports, scripts, or release metadata. — Evidence: `go.mod`, `.github/workflows/pr-test-build.yml`, `.github/workflows/release.yaml` (Documented; confidence: High)
1.1. Never commit `config.yaml` (or any custom config file) to the repository, as it contains sensitive API keys, secrets, and environment-specific settings. Use `config.example.yaml` as the template for configuration. — Evidence: `.gitignore` excludes `config.yaml` (Documented; confidence: High)
2. Prefer hot-reloadable config/auth flows over restart-only logic. The service and watcher design already support incremental auth updates and config reloads. — Evidence: `docs/sdk-watcher.md`, `sdk/cliproxy/service.go`, `sdk/cliproxy/watcher.go`, `internal/watcher/` (Documented; confidence: Medium)
3. Persist repeatable repository rules in `.osc/spec/` and keep task-specific change plans in `.osc/tasks/<task-dir>/changes/`, not in chat alone. — Evidence: `.osc/workflow.md`, `AGENTS.md`, `CLAUDE.md`, `.osc/spec/README.md` (Documented; confidence: High)
4. For non-exempt code changes, create/select a task first and write `proposal.md`, `spec.md`, and `tasks.md` before touching source files. `.osc/` files are the safe place to start. — Evidence: `.osc/workflow.md`, `AGENTS.md`, `CLAUDE.md`, `.osc/scripts/task.sh` (Documented; confidence: High)
5. Production split-proxy deployments that route to Docker-local Claude-compatible services must keep the sidecar on both the shared gateway network and the upstream service network. — Evidence: `deploy/compose.production.split-proxy.yml`, `deploy/SPLIT_PROXY_SETUP_CN.md` (Documented; confidence: High)
6. Production Grok inspection systemd units are tracked under `deploy/systemd/` and installed by `deploy/scripts/remote-deploy.sh`; keep permanent disable classes aligned with `deploy/scripts/run-grok-inspection.sh` and never put the management key value in a unit. — Evidence: `deploy/systemd/grok-inspection.service`, `deploy/systemd/grok-inspection.timer`, `deploy/scripts/remote-deploy.sh` (Documented; confidence: High; added 2026-07-24)
7. CPA-generated retry guidance may bypass upstream header passthrough only through a narrow managed-error contract; arbitrary upstream headers must remain controlled by `passthrough-headers`. — Evidence: `sdk/api/handlers/handlers.go`, `sdk/api/handlers/handlers_error_response_test.go` (Documented; confidence: High; added 2026-07-27)
8. xAI egress subscriptions, Mihomo control, lane selection, quarantine, and persistent pool state belong to the standalone `EgressProxyPool` project. CLIProxyAPI retains exact-402 classification and safe HTTP/SSE/WebSocket replay and accesses the pool only through its authenticated private API. — Evidence: `internal/runtime/executor/helps/xai_proxy_pool.go`, `internal/runtime/executor/xai_proxy_pool_executor.go`, `deploy/compose.production.xai-proxy.yml` (Documented; confidence: High; added 2026-07-27)

#### D) Testing strategy & coverage expectations
1. Treat `go build -o test-output ./cmd/server` as the minimum must-pass pull-request gate, because that is the explicitly wired PR CI check. — Evidence: `.github/workflows/pr-test-build.yml` (Documented; confidence: High)
2. When changing protocol behavior, management config handling, auth scheduling, or watcher logic, add or update focused tests in the affected package and top-level regressions when the behavior spans package boundaries. — Evidence: `test/amp_management_test.go`, `test/builtin_tools_translation_test.go`, `test/thinking_conversion_test.go`, `sdk/cliproxy/auth/*_test.go`, `internal/logging/*_test.go` (Inferred; confidence: High)
3. Release-sensitive changes must preserve CGO-disabled cross-platform builds and embedded version fields expected by GoReleaser and Docker packaging. — Evidence: `Dockerfile`, `.goreleaser.yml`, `.github/workflows/release.yaml`, `.github/workflows/docker-image.yml` (Documented; confidence: High)

#### E) Commits/PRs & review checklist
1. Before opening or updating a pull request, verify that restricted `internal/translator/**` paths were not touched unintentionally. — Evidence: `.github/workflows/pr-path-guard.yml` (Documented; confidence: High)
2. For config-management changes, confirm that comments and formatting are preserved when possible and that management writes still validate and reload cleanly. — Evidence: `internal/api/handlers/management/config_basic.go`, `internal/api/handlers/management/handler.go`, `internal/config/` (Documented; confidence: High)
3. For API-facing changes, review both client-facing compatibility and SDK embedding impact, because the server and SDK expose overlapping behavior. — Evidence: `README.md`, `docs/sdk-usage.md`, `sdk/api/handlers/`, `sdk/cliproxy/` (Inferred; confidence: High)
4. Record quality results in `.osc/quality-gate.md` instead of reporting them only in chat. — Evidence: `AGENTS.md`, `.osc/workflow.md` (Documented; confidence: High)

### Top 8 Constraints

- Constraint 1: Current source changes should stay within `.osc/tasks/07-27-extract-egress-proxy-pool` unless a new task is selected.
- Constraint 2: No source edits are allowed before `.osc/tasks/<task-dir>/changes/proposal.md`, `spec.md`, and `tasks.md` all exist or are updated, unless the user explicitly says to skip the workflow or the task type is `hotfix`/`docs`.
- Constraint 3: Required repo artifacts do not live only in chat: baseline rules go in `.osc/spec/project-spec.md`, task change packages go in `.osc/tasks/<task-dir>/changes/`, and quality results go in `.osc/quality-gate.md`.
- Constraint 4: This repository is primarily a Go proxy/server and embeddable SDK, not a bundled browser frontend. UI work in-tree normally means Bubble Tea TUI work or management asset integration.
- Constraint 5: `cmd/server` is the runnable entrypoint, and at minimum every pull request must keep `go build ./cmd/server` passing under Go `1.26`.
- Constraint 6: `internal/translator/**` is a protected boundary and cannot be changed through ordinary pull-request work.
- Constraint 7: API-facing changes must be checked against the multi-provider compatibility promise and the reusable SDK/examples that expose the same behavior.
- Constraint 8: Do not move Mihomo subscriptions or control-plane state back into CLIProxyAPI; keep the standalone service contract narrow and preserve fail-closed xAI routing.
