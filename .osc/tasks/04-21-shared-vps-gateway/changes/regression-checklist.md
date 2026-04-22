# Regression Checklist: Move CLIProxyAPI Deploy to Shared VPS Gateway

- Date: 2026-04-21
- Related: spec.md, tasks.md

## Gates (from Repo Snapshot)

- Build: `go build -o test-output ./cmd/server`
- Production compose: `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml config`
- Production compose with split-proxy: `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml config`
- Local split-proxy compose: `UPSTREAM_PROXY_HOST=proxy.example.com UPSTREAM_PROXY_PORT=3128 docker compose -f docker-compose.yml -f docker-compose.split-proxy.yml config`
- Shell syntax: `bash -n deploy/scripts/remote-deploy.sh`

## Manual checks

- On VPS, run `docker ps` and confirm only `vps-gateway-nginx` binds public `80/443`.
- On VPS, run `docker exec vps-gateway-nginx nginx -t`.
- Check `http://23.175.201.12` with `Host: api.heweili.top`; expected HTTP `301` to `https://api.heweili.top/`.
- Confirm Tailscale management binding remains `100.67.99.9:18317->8317`.
- When split-proxy is enabled, confirm `docker network inspect cli-proxy-api-proxy` lists both `kiro-rs` and `cli-proxy-split-proxy`.
- From the split-proxy container, confirm `getent hosts kiro-rs` resolves and TCP connectivity to `kiro-rs:8990` succeeds.

## Verified remote checks

- 2026-04-22: `docker exec vps-gateway-nginx nginx -t` passed; nginx reported configuration syntax is OK and test is successful.
- 2026-04-22: `docker exec cli-proxy-split-proxy getent hosts kiro-rs` passed; `kiro-rs` resolved to `172.18.0.4`.

## Edge-case re-tests

- Re-run deploy with `ENABLE_SPLIT_PROXY=true`; expected `cli-proxy-split-proxy` joins the shared network and remains running.
- Temporarily point `LOCAL_CLAUDE_NETWORK` at a missing network and run deploy; expected clear failure before compose starts.
- Stop gateway container and run deploy; expected clear failure that gateway container is missing.
- Remove gateway config dir and run deploy; expected clear failure that gateway config directory is missing.
