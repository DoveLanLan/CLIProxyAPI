# Proposal: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-01
- Owner(s): hewei
- Stakeholders: fork maintainer, production operator
- Status: Accepted
- Upstream target: `upstream/main` @ `956ce7cf` (tag `v7.2.48`)
- Last sync point: `21fad9db` (task `05-22-merge-upstream-non-docker-changes`, 2026-05-21)
- New upstream commits to absorb: `350` (`21fad9db..upstream/main`), `664` files changed

## Context / Problem

The fork's `main` has not synced with `router-for-me/CLIProxyAPI` since `21fad9db`. Upstream has since moved to `v7.2.48` and introduced major new subsystems — a plugin system (`internal/pluginhost`, `internal/pluginstore`, `sdk/pluginabi`, `sdk/pluginapi`, `sdk/pluginhost`, `sdk/pluginstore`, `examples/plugin/**`), request signature validation (`internal/signature`), safemode (`internal/safemode`), home plugins (`internal/homeplugins`), HTML sanitization and HTTP fetch helpers, plus broad translator/runtime/registry/auth/management/SDK changes (image/video handling, `gpt-image-1.5`, `disable-cooling`, `max` reasoning depth, `ResetQuota`, Codex WS↔SSE transcript replay, Gemini/Claude/Antigravity fixes, model registry updates including Claude Sonnet 5).

The fork also carries local-only deployment/ops and CPA-Manager customizations that must not be clobbered:
- Deployment/ops: `.github/workflows/**`, `Dockerfile`, `.dockerignore`, `docker-build.*`, `docker-compose*.yml`, `deploy/**`, `.goreleaser.yml`.
- CPA-Manager: `internal/config/config.go` (`DefaultPanelGitHubRepository = "https://github.com/seakee/CPA-Manager-Plus"`), `internal/managementasset/updater.go` (default release/fallback URLs), `config.example.yaml`, `deploy/compose.production.yml`, `.github/workflows/update-cpa-manager-image.yml`.
- Local protocol/runtime patches: Codex OAuth invalidated-token failover, OpenAI compat `xhigh` thinking defaults, OpenAI stream null-usage chunk handling, DeepSeek models + reasoning echo normalization, GPT-5.5 Codex support with free-tier filtering, `host.docker.internal` gateway mapping, websocket body-log growth cap, string-form system prompt preservation.

Upstream and fork both touch overlapping files (`config.example.yaml`, `Dockerfile`, `docker-compose.yml`, `README*`, `cmd/server/main.go`, `internal/managementasset/updater.go`, `go.mod/go.sum`, management handlers, runtime executors, translators), so conflicts are expected.

## Goals (Why/What)

- Bring `main` functionally to upstream `v7.2.48` (`956ce7cf`): absorb new subsystems and the translator/runtime/registry/auth/management/SDK changes.
- Keep all local deployment/ops and CPA-Manager customizations byte-for-byte on protected paths.
- Preserve local protocol/runtime behavior patches by re-applying them on top of upstream's newer structure where upstream refactored the same code.
- Keep the repo buildable and tests green: `go build -o test-output ./cmd/server`, `go test ./...`, `gofmt`, `git diff --check`, no conflict markers.

## Constraints

- Protected paths (never take upstream): `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.sh`, `docker-build.ps1`, `docker-compose*.yml`, `.env.cluster.example`, `deploy/**`, `.goreleaser.yml`.
- CPA-Manager defaults in `internal/config/config.go` and `internal/managementasset/updater.go` must stay local (`seakee/CPA-Manager-Plus`).
- Do not use a plain `git merge upstream/main` — upstream lacks local `.osc` state and would delete local workflow/task files; the prior syncs used patch-apply with three-way fallback, and this task continues that approach.
- `internal/translator/**` is protected by CI for ordinary PRs, but is in scope here because the upstream sync spans broader protocol/runtime changes (same precedent as `05-22-merge-upstream-non-docker-changes`).
- No secrets or environment-specific config committed; `config.yaml` stays ignored.
- Stay local — pushing to remotes and production deployment are separate operations.

## Non-goals

- Do not redesign upstream changes during the sync.
- Do not adopt upstream's deployment/CI/Docker changes (explicit exclusion).
- Do not change local runtime config files such as `config.yaml`, `auths/`, `logs/`.
- Do not create a merge commit unless explicitly requested (follow prior sync precedent: working-tree sync).
- Do not review or rewrite the plugin system's design — take upstream as-is.

## Proposed Approach (high-level)

Generate an upstream-only patch from `21fad9db..upstream/main` excluding protected paths, apply it to the current working tree with three-way fallback (`git merge-tree` / `git apply --3way` or an explicit `git checkout upstream/main -- <path>` for clean additions), then:

1. Restore protected paths from `HEAD` (deploy/ops/CI/Docker).
2. Re-apply CPA-Manager defaults and local behavior patches onto upstream's newer structure (Codex invalidated-token failover, xhigh thinking defaults, null-usage chunks, DeepSeek models/reasoning echo, GPT-5.5 Codex free-tier filter, `host.docker.internal` mapping, websocket body-log cap, string system prompt).
3. Resolve remaining conflicts module-by-module, generally favoring upstream functional code unless it conflicts with a fork-specific requirement documented above.
4. Run quality gates and record closure artifacts.

Conflict resolution priority:
- Protected path → keep local `HEAD`.
- CPA-Manager defaults → keep local values.
- Local behavior patch (Codex failover, xhigh, null usage, DeepSeek, GPT-5.5, etc.) → keep local behavior, adapt to upstream's new call sites/structure.
- Everything else (new subsystems, translator/runtime/registry/auth/management/SDK) → take upstream.

## Risks & Mitigations

- Risk: Upstream refactored files that carry local behavior patches (e.g. `internal/runtime/executor/*`, `sdk/cliproxy/auth/*`, `internal/api/handlers/management/*`), so local patches may not apply cleanly.
  - Mitigation: Re-apply patches by intent against upstream's new structure, run the focused tests for each local behavior to confirm, fix call sites until green.
- Risk: Upstream's plugin system adds new config fields, management endpoints, and registry entries that may interact with local config or CPA-Manager management asset logic.
  - Mitigation: Take upstream's config/registry/management changes, then re-assert CPA-Manager defaults and run management-handler tests.
- Risk: `internal/translator/**` changes trip the protected-path PR guard.
  - Mitigation: Document that translator changes ride with the broader upstream sync (precedent: `05-22-merge-upstream-non-docker-changes`); this lands on `main` directly, not via a translator-only PR.
- Risk: Large diff makes review and rollback hard.
  - Mitigation: Do the sync on a temporary branch first, keep rollback point recorded, write a per-area change summary.
- Risk: Upstream may have removed or renamed providers/fields the fork still references (e.g. prior Qwen/iFlow removal).
  - Mitigation: Accept upstream's removals; verify build/tests catch any dangling references.

## Open Questions (max 3)

- None. The user scoped this as "merge upstream v7.2.48 + protect deploy/CPA-Manager," matching the established sync precedent.
