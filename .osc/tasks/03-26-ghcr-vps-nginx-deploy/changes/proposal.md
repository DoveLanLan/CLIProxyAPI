# Proposal: Set Up GHCR GitHub Actions Deployment To HK VPS

- Date: 2026-03-26
- Owner(s): hewei
- Stakeholders: hewei, operators of `api.heweili.top`, CLIProxyAPI clients using the hosted proxy
- Status: Proposed

## Context / Problem

The repository already contains Docker packaging and GitHub Actions workflows, but the current automation is not suitable for the user's forked production deployment. The existing Docker workflow publishes only on git tags, pushes to the upstream Docker Hub namespace, and assumes a broader multi-arch release flow than the user's immediate target requires. The repository also lacks a production-ready Compose stack with Nginx reverse proxying, domain-based TLS termination, and a clear separation between public proxy endpoints and private management surfaces.

The user wants to deploy this fork to a Hong Kong VPS at `23.175.201.12` behind `api.heweili.top`, using GitHub Actions as the automated delivery path, GHCR as the image registry, Docker Compose plus Nginx on the server, and Cloudflare-managed DNS/TLS. The user also wants `config.yaml`, `auths/`, and `logs/` to remain on the VPS rather than in repository or GitHub secrets.

## Goals (Why/What)

- Publish production images for this fork to GHCR from GitHub Actions.
- Deploy the hosted proxy automatically to the Ubuntu x86_64 VPS using SSH and Docker Compose.
- Terminate public HTTPS at Nginx for `api.heweili.top` while keeping the app container off direct public host-port exposure.
- Keep the management UI and management API off the public internet for the first version.
- Preserve existing release packaging where useful while separating production deployment concerns from upstream release-only workflows.
- Leave an auditable task package describing the deployment design, workflow changes, and rollback path.

## Constraints

- Must follow the `osc` change workflow before editing non-`.osc/` files.
- Must preserve the repo's build metadata injection and `go build ./cmd/server` CI baseline.
- Must target `linux/amd64` first because the destination VPS is `x86_64`.
- Must use GHCR instead of the upstream Docker Hub repository.
- Must assume the deployment SSH user is `root` on `23.175.201.12:22`.
- Must keep runtime secrets and persistent state on the VPS, not in tracked files.
- Must not expose the management panel or `/v0/management` publicly in the first version.
- Must work cleanly with Cloudflare in front of the origin and a Cloudflare Origin CA certificate on Nginx.

## Non-goals

- Reworking application internals unrelated to deployment, such as translators or provider logic.
- Adding a new first-party web admin frontend.
- Publishing arm64 production images in the first pass.
- Implementing zero-downtime blue/green deployment.
- Exposing or redesigning remote management flows in this task.

## Proposed Approach (high-level)

Create a dedicated production deployment package in the repository that includes a production Compose definition, Nginx configuration, and environment templates for the forked deployment. Replace the current Docker publishing workflow with a GHCR-oriented image pipeline that publishes deterministic amd64 tags for the fork, and add a production deployment workflow that connects to the VPS over SSH, prepares the deployment directory, refreshes deployment files, pulls the new image, and recreates the Compose stack.

Use Nginx as the only public entrypoint on `80/443`, reverse proxying to the internal app container on port `8317` with streaming- and WebSocket-safe settings. Explicitly block management paths in Nginx for the first release. Keep `config.yaml`, `auths/`, `logs/`, and TLS certificate files on the server, and document Cloudflare-side prerequisites: proxied DNS record, `Full (strict)` TLS mode, and Origin CA certificate/key placement on the origin.

## Risks & Mitigations

- Risk: GHCR publication succeeds but the package remains private, causing anonymous pulls on the VPS to fail.
  - Mitigation: document and verify that the GHCR package visibility is set to public after first publish; fall back to registry login only if that is intentionally not done.
- Risk: direct SSH deployment from GitHub Actions increases blast radius if the deployment key leaks or the workflow is tampered with.
  - Mitigation: scope the key to deployment only, pin `known_hosts`, keep branch protections on `main`, and isolate deployment secrets in a production environment.
- Risk: Nginx proxy settings may break streaming responses or WebSocket upgrades used by clients.
  - Mitigation: configure HTTP/1.1 proxying, disable buffering for proxy responses, and forward `Upgrade` / `Connection` headers explicitly.
- Risk: the existing local/dev Compose file could be unintentionally repurposed for production and keep exposing extra host ports.
  - Mitigation: add a dedicated production Compose file instead of layering production on the current local Compose setup.
- Risk: management routes become reachable accidentally through the production domain.
  - Mitigation: disable the control-panel route in app config where appropriate and block `/management.html` plus `/v0/management/` at Nginx.

## Open Questions (max 3)

- None. The remaining deployment inputs are concrete enough to proceed with implementation.
