# Regression Checklist: Fix CPA-Manager Monitoring Load

- Date: 2026-05-22
- Related: spec.md, tasks.md

## Gates

- Compose render: `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test docker compose -f deploy/compose.production.yml config`
- Diff hygiene: `git diff --check`
- Build: `go build -o test-output ./cmd/server && rm test-output`

## Manual VPS Checks

- Restart only the CPA-Manager service after deploying the compose change.
- Open `http://100.67.99.9:18318/management.html#/monitoring`.
- Confirm `/status` returns quickly and reports the collector as running.
- Confirm `/v0/management/usage` returns before the frontend timeout and the loading overlay disappears.
- If it still times out, set `CPA_MANAGER_USAGE_QUERY_LIMIT=200` or `100` in `/opt/cliproxyapi/.env` and restart only `cpa-manager`.

## Security Checks

- Do not paste management keys, bearer tokens, auth files, or local `config.yaml` content into logs or issues.
- Confirm public Cloudflare routes still block `/management.html` and `/v0/management`.
