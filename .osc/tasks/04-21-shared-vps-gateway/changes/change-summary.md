# Change Summary: Move CLIProxyAPI Deploy to Shared VPS Gateway

- Date: 2026-04-21
- Owner(s): hewei
- Related: spec.md, tasks.md

## What changed

- Removed the production nginx service from `deploy/compose.production.yml`.
- Switched the production `proxy` network to external `${GATEWAY_NETWORK:-vps-gateway}`.
- Added gateway env defaults to `deploy/.env.example`.
- Updated `deploy/scripts/remote-deploy.sh` to install `api.heweili.top.conf` into the shared gateway and reload `vps-gateway-nginx`.
- Updated `api.heweili.top.conf` to use Docker DNS resolver variables so gateway config remains reloadable.
- Updated deployment docs to describe the shared gateway topology.

## Why

The VPS now uses `/opt/vps-gateway` as the only public nginx entrypoint. Without this change, future CLIProxyAPI deployments could recreate `cli-proxy-api-nginx` and conflict with `vps-gateway-nginx` on public ports `80/443`.

## Notable decisions

- CLIProxyAPI continues to own the `api.heweili.top` route config, but the shared gateway owns the nginx container.
- Tailscale management port behavior remains unchanged.
- Split-proxy continues to work as a compose override on the shared gateway network.
- Server-side `.env` remains the source of deployment/runtime secrets.
