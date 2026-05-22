# Regression Checklist: Make CPA-Manager Image Configurable

- Date: 2026-05-22
- Related: spec.md, tasks.md

## Gates

- Default compose render: `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test docker compose -f deploy/compose.production.yml config`
- Override compose render: `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test CPA_MANAGER_IMAGE=ghcr.io/example/cpa-manager:sha-test docker compose -f deploy/compose.production.yml config`
- Diff hygiene: `git diff --check`
- Build: `go build -o test-output ./cmd/server && rm test-output`

## Results

- Default compose render keeps `image: seakee/cpa-manager:latest`.
- Override compose render uses `image: ghcr.io/example/cpa-manager:sha-test`.
- `git diff --check` passed.
- `go build -o test-output ./cmd/server && rm test-output` passed.

## Manual VPS Checks

- Set `CPA_MANAGER_IMAGE` in `/opt/cliproxyapi/.env` only after the fork image is published.
- Run `docker compose -f compose.production.yml config` on the VPS and confirm the rendered CPA-Manager image.
- Restart only the CPA-Manager service and confirm `http://<tailscale-ip>:18318/management.html#/monitoring` renders.
