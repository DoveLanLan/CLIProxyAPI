# Regression Checklist: Fix split-proxy Squid logging startup failure

- Date: 2026-04-09
- Related: `proposal.md`, `spec.md`, `tasks.md`

## Gates (from Repo Snapshot)

- Build: `go build -o /tmp/cli-proxy-api-check ./cmd/server`
- Tests: no repo CI test job targets deploy assets beyond compile; rely on targeted deploy-file validation for this bugfix
- Lint/format: not explicitly configured for deploy shell/docs files in repo evidence
- Other:
  - `bash -n deploy/split-proxy/start.sh`
  - `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml config`
  - `UPSTREAM_PROXY_HOST=proxy.example.com UPSTREAM_PROXY_PORT=3128 docker compose -f docker-compose.yml -f docker-compose.split-proxy.yml config`

## Manual checks (if applicable)

- Redeploy on a server with `ENABLE_SPLIT_PROXY=true` and confirm `cli-proxy-split-proxy` stays in `Up` state. (Expected: no restart loop)
- Verify `/opt/cliproxyapi/data/logs/split-proxy/access.log` and `/opt/cliproxyapi/data/logs/split-proxy/cache.log` are created after startup. (Expected: files exist and contain Squid output)
- Confirm `cli-proxy-api` can still reach `http://split-proxy:3128` after the sidecar restart. (Expected: no proxy connection regression)

## Edge-case re-tests

- First boot when `/opt/cliproxyapi/data/logs/split-proxy/` does not exist yet.
- Redeploy when the log directory exists but is owned by `root`.
