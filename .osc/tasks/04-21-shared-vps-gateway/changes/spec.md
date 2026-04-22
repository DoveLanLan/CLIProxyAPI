# Spec: Move CLIProxyAPI Deploy to Shared VPS Gateway

- Date: 2026-04-21
- Owner(s): hewei
- Related: proposal.md, tasks.md

## Repo Snapshot

- App stack: Go server entrypoint under `cmd/server`, Docker image built by `Dockerfile`, production runtime defined under `deploy/`.
- Current production deploy: `deploy/compose.production.yml` owns `cli-proxy-api` plus nginx; `deploy/scripts/remote-deploy.sh` validates config/certs and runs Docker Compose.
- CI/deploy: `.github/workflows/docker-image.yml` publishes GHCR image; `.github/workflows/deploy-production.yml` uploads `deploy/` to `/opt/cliproxyapi` and runs `scripts/remote-deploy.sh`.
- Required gates: `go build ./cmd/server`; production compose config validation; shell syntax validation.

## Scope

### In scope

- Remove production nginx service from CLIProxyAPI compose.
- Add shared gateway network variables to env template and docs.
- Update remote deploy script to install/reload gateway route config.
- Attach production split-proxy to the external Docker network that contains local Claude-compatible services such as `kiro-rs`.
- Document `LOCAL_CLAUDE_NETWORK` and direct-bypass host defaults for split-proxy.
- Update deployment README to describe the gateway topology.

### Out of scope

- Go server implementation changes.
- GitHub workflow trigger or SSH behavior changes.
- Gateway stack source repository creation.
- Runtime `.env` secrets.

## Acceptance Criteria (testable)

1. `deploy/compose.production.yml` has no nginx service. (Verify: inspect file)
2. `deploy/compose.production.yml` uses external network `${GATEWAY_NETWORK:-vps-gateway}`. (Verify: compose config)
3. `deploy/scripts/remote-deploy.sh` copies `api.heweili.top.conf` to `${GATEWAY_ROOT:-/opt/vps-gateway}/nginx/conf.d/` and reloads `${GATEWAY_CONTAINER:-vps-gateway-nginx}`. (Verify: inspect script)
4. `deploy/README.md` documents CLIProxyAPI as an app behind the shared gateway. (Verify: inspect docs)
5. When split-proxy is enabled, `cli-proxy-split-proxy` joins both `${GATEWAY_NETWORK:-vps-gateway}` and `${LOCAL_CLAUDE_NETWORK:-cli-proxy-api-proxy}`. (Verify: production split-proxy compose config)
6. The deploy script validates that `${LOCAL_CLAUDE_NETWORK:-cli-proxy-api-proxy}` exists before starting split-proxy. (Verify: inspect script / shell syntax)
7. Local checks pass. (Verify: `go build -o test-output ./cmd/server`, `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml config`, `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml config`, `bash -n deploy/scripts/remote-deploy.sh`)

## Behavior / Requirements

- CLIProxyAPI remains reachable to the gateway as `cli-proxy-api:8317`.
- The public nginx route still denies public management UI/API paths.
- Tailscale management access remains bound to `${TAILSCALE_BIND_IP}:${TAILSCALE_MANAGEMENT_PORT}`.
- If split-proxy is enabled, split-proxy joins the same external network as the app and the external local-Claude network used by services such as `kiro-rs`.
- The deploy script creates the external Docker network if it is missing, but requires the gateway config directory and gateway container to exist before nginx install/reload.
- The deploy script does not auto-create the local-Claude network because creating an empty network would not connect the upstream service; it must already exist and include the local Claude-compatible container.

## Edge Cases

- If gateway stack is missing, deploy should fail after app compose restart with a clear operational error.
- If split-proxy is enabled and the local-Claude network is missing, deploy should fail before compose starts with a clear operational error.
- If route config is invalid, `docker exec vps-gateway-nginx nginx -t` should fail before reload.
- If old `PUBLIC_BIND_IP` remains in server `.env`, it is ignored by app compose and harmless.
- If the local upstream service uses an older container name such as `kirors-kiro`, direct bypass should remain configurable through `DIRECT_DOMAINS`.

## Compatibility Notes

- Backwards compatibility: API service, management port, config paths, auth paths, and split-proxy runtime behavior remain unchanged.
- Data/migrations: no data migrations.
- Config/flags: new optional env vars `GATEWAY_NETWORK`, `GATEWAY_ROOT`, `GATEWAY_CONTAINER`, and `LOCAL_CLAUDE_NETWORK`.

## API/UX Decisions (if applicable)

- Public API domain remains `api.heweili.top`.
- Gateway stack owns public `80/443`; CLIProxyAPI does not.
