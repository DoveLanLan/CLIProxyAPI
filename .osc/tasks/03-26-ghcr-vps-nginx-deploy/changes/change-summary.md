# Change Summary: Set Up GHCR GitHub Actions Deployment To HK VPS

- Date: 2026-03-26
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`, `tasks.md`

## What changed

- Added a task-scoped deployment proposal/spec/tasks package for the GHCR + VPS + Nginx + Cloudflare Origin CA rollout.
- Added a dedicated `deploy/` production deployment package with:
  - `compose.production.yml`
  - Nginx reverse-proxy config for `api.heweili.top`
  - remote deployment script
  - server/bootstrap documentation
- Replaced the old Docker Hub publishing workflow with a GHCR amd64 publishing workflow for the fork.
- Added a new production deployment workflow that deploys the published image to `root@23.175.201.12:/opt/cliproxyapi` over SSH.
- Kept management endpoints off the public domain by blocking `/management.html` and `/v0/management/` at Nginx while exposing a dedicated Tailscale-only management port.

## Why

The upstream repository's release automation was aimed at tag-based public releases and an upstream Docker Hub namespace, not at the user's forked production deployment. The new change set gives the fork a direct path from `main` to a reproducible VPS deployment while keeping live config, auth files, logs, and TLS private material on the server.

## Notable decisions

- Chose GHCR over Docker Hub to align with the user's fork and avoid maintaining separate Docker Hub credentials.
- Chose amd64-only image publication because the target VPS is `x86_64`; multi-arch can be added later if needed.
- Chose a dedicated production Compose stack instead of modifying the root local/dev Compose file.
- Chose Cloudflare Origin CA for the first HTTPS setup because the domain is already behind Cloudflare and the user wanted the simpler operational path.
- Chose split management access: public domain remains blocked, while a dedicated Tailscale-bound port supports private admin access.
