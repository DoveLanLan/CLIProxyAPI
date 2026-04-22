# Journal 3: Move CLIProxyAPI to Shared VPS Gateway

- Date: 2026-04-21
- Task: `.osc/tasks/04-21-shared-vps-gateway`

## Decisions

- CLIProxyAPI no longer owns a public nginx container.
- The public entrypoint is the shared `/opt/vps-gateway` stack and container `vps-gateway-nginx`.
- CLIProxyAPI still owns and deploys its `api.heweili.top` route config into the gateway config directory.
- Tailscale management binding remains unchanged.

## Changes

- Removed nginx from `deploy/compose.production.yml`.
- Switched production network to external `${GATEWAY_NETWORK:-vps-gateway}`.
- Updated `remote-deploy.sh` to create/use gateway network, install route config into gateway, and reload `vps-gateway-nginx`.
- Updated `api.heweili.top.conf` to use Docker DNS resolver variables.
- Updated deployment docs and `.env.example`.
- Synced updated deploy assets to `bytevirt` and ran remote deploy successfully.

## Verification

- Passed Go build.
- Passed production compose config with and without split-proxy.
- Passed deploy script shell syntax and YAML parse checks.
- Passed remote gateway nginx test.
- `api.heweili.top` returns HTTP `301` through the new gateway.

## Risks / Follow-ups

- GitHub Actions deploy workflow should be validated after commit/push.
- Gateway stack is now a shared operational dependency for public access.
- Certificates still live under `/opt/cliproxyapi/certs` and are mounted by the gateway.

## Rollback

- Restore `/opt/cliproxyapi/compose.production.yml.bak.20260421-063148` on the VPS to bring back the old CLIProxyAPI-owned nginx stack.
- Revert repo deploy asset changes if needed.
