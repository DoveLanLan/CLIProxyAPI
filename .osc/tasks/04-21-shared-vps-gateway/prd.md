# PRD: Move CLIProxyAPI production deploy to shared VPS gateway

## Problem

`CLIProxyAPI` currently owns the production nginx container in `deploy/compose.production.yml`. The VPS has been moved to a shared `vps-gateway-nginx` entrypoint so multiple apps can share the same public `80/443` bindings. If this repository keeps deploying its own nginx service, future `deploy-production` runs can recreate `cli-proxy-api-nginx` and conflict with the shared gateway.

## Goals

- Remove the production nginx service from the CLIProxyAPI app compose stack.
- Connect `cli-proxy-api` and optional `split-proxy` to the external `vps-gateway` Docker network.
- Keep `api.heweili.top` nginx config in the repository as the app-owned gateway route config.
- Update `remote-deploy.sh` to install the route config into `/opt/vps-gateway/nginx/conf.d/` and reload `vps-gateway-nginx`.
- Keep Tailscale management port behavior unchanged.

## Non-goals

- Do not change the CLIProxyAPI Go server behavior.
- Do not alter API route compatibility or management route policy.
- Do not create a new gateway repository in this change.
- Do not change split-proxy behavior except for using the shared Docker network.

## Acceptance criteria

- `deploy/compose.production.yml` contains only the `cli-proxy-api` app service and no nginx service.
- Production compose declares `proxy` as external network `${GATEWAY_NETWORK:-vps-gateway}`.
- `remote-deploy.sh` creates/uses the gateway network, starts app services, installs `api.heweili.top.conf` into the shared gateway conf dir, validates gateway nginx, and reloads it.
- Deployment docs no longer describe CLIProxyAPI as owning public nginx.
- Local compose config validation and Go build pass.
