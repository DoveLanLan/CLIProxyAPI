# Regression Checklist: Configure Custom Providers And Docker Compose Startup

- Date: 2026-03-09
- Related: `spec.md`, `tasks.md`

## Gates (from Repo Snapshot)

- Compose config validation: `docker compose -f docker-compose.yml -f docker-compose.local.yml config`
- Container startup: `docker compose -f docker-compose.yml -f docker-compose.local.yml up -d`
- Runtime status: `docker compose -f docker-compose.yml -f docker-compose.local.yml ps`
- Root endpoint health: `curl http://127.0.0.1:8317/`
- Model registration sanity: `curl -H 'Authorization: Bearer <local-proxy-key>' http://127.0.0.1:8317/v1/models`
- Build gate from repo CI: not run for this change set because no tracked runtime source code was modified

## Manual checks (if applicable)

- Verify the root endpoint returns the CLI Proxy API server banner.
- Verify `/v1/models` includes prefixed models for `nih`, `linuxdo`, `claude-local`, `claude-fc`, and `claude-sf`.
- Verify requests to prefixed models use the intended upstream instead of ambiguous unprefixed routing.
- Verify `claude-sf/minimax-m2.5` is listed after config reload or restart.

## Edge-case re-tests

- If the host-local Claude upstream on port `8990` is down, verify only `claude-local/*` requests fail; the rest of the proxy should remain available.
- If Docker host ports other than `8317` are busy, confirm the local compose override still starts cleanly.
