# Spec: Fix split-proxy Squid logging startup failure

- Date: 2026-04-09
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## Repo Snapshot (from step 0)

- Modules/components:
  - `deploy/` holds the production compose files, deploy scripts, and split-proxy sidecar assets.
  - `.github/workflows/deploy-production.yml` uploads only `deploy/` contents to the server and runs `scripts/remote-deploy.sh`.
  - `deploy/scripts/remote-deploy.sh` conditionally enables `compose.production.split-proxy.yml` via `ENABLE_SPLIT_PROXY=true`.
- Toolchains:
  - Build: `go build -o test-output ./cmd/server` from `.github/workflows/pr-test-build.yml`
  - Runtime validation: deploy shell scripts plus compose files under `deploy/`
  - Quality: shell syntax and compose rendering are the strongest repo-backed checks for this change area
- Confidence: High
- Evidence:
  - `deploy/split-proxy/start.sh`
  - `deploy/compose.production.split-proxy.yml`
  - `docker-compose.split-proxy.yml`
  - `deploy/scripts/remote-deploy.sh`
  - `deploy/SPLIT_PROXY_SETUP_CN.md`
  - `.github/workflows/deploy-production.yml`

## Scope
### In scope

- Change the generated Squid log targets away from `/dev/stdout` and `/dev/stderr`.
- Ensure the split-proxy container creates writable runtime log/spool directories before starting Squid.
- Persist split-proxy logs through compose-mounted host directories for both local and production overrides.
- Update split-proxy deployment docs and troubleshooting guidance.

### Out of scope

- Changing proxy ACLs, peer selection, or bypass host/CIDR defaults.
- Changing the main server image, Go code, or deploy workflow triggers.
- Adding log rotation or external log shipping.

## Acceptance Criteria (testable)

1. `deploy/split-proxy/start.sh` no longer renders Squid log outputs to `/dev/stdout` or `/dev/stderr`. (Verify: inspect generated script source)
2. The split-proxy compose overrides mount a host log directory into the Squid log path used by the startup script. (Verify: inspect `deploy/compose.production.split-proxy.yml` and `docker-compose.split-proxy.yml`)
3. The startup script creates/chowns the Squid log and spool directories before `exec squid`, so the `proxy` runtime user can write logs. (Verify: inspect `deploy/split-proxy/start.sh`)
4. The deployment docs explicitly tell operators where split-proxy logs now live and how to inspect them after deployment. (Verify: inspect `deploy/SPLIT_PROXY_SETUP_CN.md` and `deploy/split-proxy/README.md`)

## Behavior / Requirements

The split-proxy sidecar must continue to render a minimal Squid config using the existing env vars:

- `UPSTREAM_PROXY_HOST`
- `UPSTREAM_PROXY_PORT`
- `UPSTREAM_PROXY_LOGIN`
- `DIRECT_DOMAINS`
- `DIRECT_CIDRS`
- `SQUID_HTTP_PORT`

The only behavioral change is log destination handling:

- `access_log` and `cache_log` must use writable file paths.
- `start.sh` must prepare those paths before launching Squid.
- The compose overrides must persist those logs on the host so they survive container restarts and remain inspectable.

## Edge Cases

- First deployment onto a host where `data/logs/split-proxy` does not exist yet.
- Redeploy onto a host where the log directory exists but is root-owned from a previous run.
- Local compose users and production deploy users must both get the same runtime behavior despite different relative volume paths.
- Operators following older docs must still have a clear recovery path once the docs are updated.

## Compatibility Notes

- Backwards compatibility: existing split-proxy env vars and `proxy-url: http://split-proxy:3128` remain unchanged.
- Data/migrations: none; this adds log file persistence only.
- Config/flags: no new env vars or runtime flags.

## API/UX Decisions (if applicable)

- Inputs/outputs: no public API changes.
- States/errors: startup should no longer fail solely because Squid cannot open `/dev/stdout` or `/dev/stderr`.
- Telemetry/logging: split-proxy logs move from container stdio to persisted files under the mounted log directory.
- Accessibility/i18n (if UI): not applicable.
