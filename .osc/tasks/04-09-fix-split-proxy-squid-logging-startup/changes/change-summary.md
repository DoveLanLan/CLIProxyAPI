# Change Summary: Fix split-proxy Squid logging startup failure

- Date: 2026-04-09
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`, `tasks.md`

## What changed

- Updated `deploy/split-proxy/start.sh` so Squid writes logs to `/var/log/squid/access.log` and `/var/log/squid/cache.log` instead of `/dev/stdout` and `/dev/stderr`.
- Added runtime directory preparation and ownership fixes for `/var/log/squid` and `/var/spool/squid` before Squid starts.
- Mounted a host log directory into `/var/log/squid` for both production and local split-proxy compose overrides.
- Updated the production deployment docs and split-proxy README to point operators at the persisted log files.

## Why

The current split-proxy container crashed because Squid could not open `/dev/stdout` and `/dev/stderr` after switching to the `proxy` runtime user. Persisted writable log files remove that startup failure while keeping operator-visible logs available on the host.

## Notable decisions

- The fix stays deploy-only and does not modify proxy routing, upstream peer logic, or GitHub Actions flow.
- Logs now live in mounted files rather than container stdio because the deployed Squid image rejects the previous stdio targets under its effective runtime user.
