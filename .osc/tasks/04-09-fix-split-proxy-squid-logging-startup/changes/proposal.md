# Proposal: Fix split-proxy Squid logging startup failure

- Date: 2026-04-09
- Owner(s): hewei
- Stakeholders: deploy operators, production runtime owners
- Status: Proposed

## Context / Problem

The production `split-proxy` sidecar currently renders Squid log targets as `stdio:/dev/stdout` and `stdio:/dev/stderr`.
On the deployed `ubuntu/squid` image this causes startup to fail with:

- `FATAL: Cannot open '/dev/stdout' for writing.`
- `The parent directory must be writeable by the user 'proxy'`

As a result, the `cli-proxy-split-proxy` container restarts continuously and any deployment with `ENABLE_SPLIT_PROXY=true` is broken.

## Goals (Why/What)

- Make the `split-proxy` sidecar start reliably on production hosts that use the current Squid image.
- Keep split-proxy logs available for operators after the container no longer writes directly to `/dev/stdout`.
- Update deployment docs so operators know where to inspect logs and how to recover.

## Constraints

- Keep the existing split-proxy routing behavior, env vars, and upstream proxy wiring unchanged.
- Stay within deploy-side files only; do not change the main Go server runtime.
- Preserve compatibility for both local compose usage and production compose usage.

## Non-goals

- Changing split-proxy ACL rules, upstream peer behavior, or Docker networking.
- Changing GitHub Actions workflow shape beyond the deploy assets already shipped.

## Proposed Approach (high-level)

Stop rendering Squid logs to `/dev/stdout` and `/dev/stderr`. Instead, create writable log files under `/var/log/squid`, ensure ownership matches the Squid runtime user, and mount that directory from the host in both split-proxy compose overrides so logs persist. Update the split-proxy docs and server setup guide to point operators at the persisted log files.

## Risks & Mitigations

- Risk: Host-mounted log directories may have the wrong ownership on first boot.
  - Mitigation: create and `chown` the Squid log and spool directories in `start.sh` before `exec squid`.
- Risk: Operators may still rely on `docker compose logs -f split-proxy`.
  - Mitigation: update docs with the new log location and example inspection commands.
- Risk: Existing deployments may contain the old `start.sh` until the next asset sync.
  - Mitigation: document a server-side hotfix path in the setup guide and final response.

## Open Questions (max 3)

- None.
