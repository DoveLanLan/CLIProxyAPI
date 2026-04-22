# Proposal: Move CLIProxyAPI Deploy to Shared VPS Gateway

- Date: 2026-04-21
- Owner(s): hewei
- Stakeholders: VPS operator, CLIProxyAPI maintainer
- Status: Accepted

## Context / Problem

The VPS now uses an independent `vps-gateway-nginx` container as the only public `80/443` entrypoint. `CLIProxyAPI` still has production deploy files that define and deploy `cli-proxy-api-nginx`, which would recreate the old nginx container and conflict with the shared gateway on future deploys.

## Goals (Why/What)

- Make CLIProxyAPI an app stack behind the shared gateway instead of a public nginx owner.
- Keep `api.heweili.top` route config versioned with the CLIProxyAPI repo.
- Ensure future `deploy-production` workflow runs do not recreate `cli-proxy-api-nginx`.
- Preserve Tailscale-bound management access and split-proxy support.

## Constraints

- Shared gateway network defaults to `vps-gateway`.
- Gateway root defaults to `/opt/vps-gateway`.
- Gateway container defaults to `vps-gateway-nginx`.
- Do not change Go app code or API behavior.
- Runtime secrets remain on the VPS `.env`.

## Non-goals

- No new gateway repository in this change.
- No HTTPS certificate migration beyond continuing to use gateway-mounted certs.
- No split-proxy functional redesign.

## Proposed Approach (high-level)

Remove nginx from `deploy/compose.production.yml`, declare the app network as external `vps-gateway`, and update the deploy script to copy `deploy/nginx/conf.d/api.heweili.top.conf` into the shared gateway config directory followed by `docker exec vps-gateway-nginx nginx -t` and reload. Update docs and env template to describe the shared gateway contract.

## Risks & Mitigations

- Risk: gateway stack missing during app deploy.
  - Mitigation: deploy script fails with a clear error if `/opt/vps-gateway/nginx/conf.d` or `vps-gateway-nginx` is missing.
- Risk: nginx route config references unavailable app service.
  - Mitigation: app compose is started before gateway reload.
- Risk: future deploys break management access.
  - Mitigation: leave Tailscale management port mapping unchanged.

## Open Questions (max 3)

- None. VPS runtime has already been migrated to `vps-gateway-nginx`.
