# Spec: Set Up GHCR GitHub Actions Deployment To HK VPS

- Date: 2026-03-26
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## Repo Snapshot (from step 0)

- Modules/components:
  - `cmd/server` runnable server entrypoint
  - `internal/api`, `internal/config`, `internal/managementasset` runtime and management surfaces
  - `.github/workflows` CI/release automation
  - Docker packaging via `Dockerfile`, `docker-compose.yml`, `docker-compose.example.yml`
- Toolchains:
  - Build: Go modules, `go build ./cmd/server`, Docker multi-stage build
  - CI: PR compile check, translator path guard, release and Docker publish workflows
  - Runtime: YAML config mounted into the container, file-backed auth/log directories
- Confidence: High
- Evidence: `cmd/server/main.go`, `Dockerfile`, `config.example.yaml`, `docker-compose.yml`, `.github/workflows/pr-test-build.yml`, `.github/workflows/pr-path-guard.yml`, `.github/workflows/release.yaml`, `.github/workflows/docker-image.yml`

## Scope

### In scope

- Add or update GitHub Actions workflows for:
  - GHCR image publishing for this fork
  - production deployment to the VPS over SSH
- Add production deployment artifacts for:
  - Compose stack
  - Nginx reverse proxy
  - deployment environment template / documentation
- Keep release packaging and source build metadata intact.
- Limit the first production deployment to `linux/amd64`.
- Keep management surfaces private in the first deployment design.
- Document Cloudflare Origin CA prerequisites and server-side file expectations.

### Out of scope

- Adding remote/public management-panel support.
- Replacing Nginx with another reverse proxy.
- Adding arm64 production deployment.
- Reworking provider configs, auth flows, or runtime business logic.
- Storing live production `config.yaml` or auth files in the repository.

## Acceptance Criteria (testable)

1. The repository contains a production-oriented deployment workflow that can publish an amd64 image for this fork to GHCR. (Verify: inspect workflow + `act`-style logic review or GitHub workflow syntax validation)
2. The published image name no longer references the upstream Docker Hub namespace and instead resolves to the fork's GHCR namespace. (Verify: inspect workflow tags / image variables)
3. The repository contains a production deployment workflow that can SSH to the VPS, ensure the deployment directory exists, refresh deployment files, and run `docker compose pull` / `up -d`. (Verify: inspect workflow steps and remote deploy script/commands)
4. The repository contains a production Compose definition that exposes only Nginx publicly while the app container remains internal to the Docker network. (Verify: inspect Compose `ports`, `expose`, and service definitions)
5. The repository contains Nginx config for `api.heweili.top` that proxies the CLIProxyAPI service correctly for standard HTTP, streaming responses, and `/v1/ws`. (Verify: inspect Nginx config for `proxy_http_version`, buffering, and upgrade headers)
6. Management UI/API routes are not exposed publicly by the production proxy configuration, but remain privately reachable over a dedicated Tailscale-bound port. (Verify: inspect Nginx deny rules, Compose port binding, and deployment docs)
7. The deployment artifacts explicitly document Cloudflare-side setup for Origin CA and `Full (strict)` mode. (Verify: inspect deployment docs / env template / final notes)

## Behavior / Requirements

- The GHCR image tags should be deterministic and traceable to source, at minimum including a commit-SHA-oriented tag and a stable branch tag.
- The workflow should use the current repository/fork identity rather than hardcoded upstream registry values.
- Production deployment should be safe to rerun and should create the deployment directory if missing.
- Public traffic should terminate at Nginx on the VPS, with Nginx forwarding to the internal app service on port `8317`.
- Nginx should preserve host/proto headers and support long-lived AI streaming responses without proxy buffering issues.
- Production Compose should not publish callback helper ports such as `8085`, `1455`, `54545`, `51121`, or `11451` to the public host.
- Production runtime state should live on the server under persistent paths for `config.yaml`, `auths/`, `logs/`, and certificate files.
- The deployment design should keep management off the public internet while allowing access over a dedicated Tailscale-bound host port and optional SSH tunnel fallback.

## Edge Cases

- The GHCR package may default to private after first publish, preventing anonymous server pulls.
- GitHub Actions may deploy before the Cloudflare Origin CA cert and key are present on the VPS.
- The server may not yet contain `config.yaml` or `auths/`, causing the app to enter standby or incomplete startup behavior.
- SSH host key rotation would break strict host verification until `known_hosts` is updated.
- Proxy buffering or default timeouts may break long streaming responses if not explicitly configured.
- Root-owned deployment paths under `/opt` will work, but later switching to a non-root deploy user would require directory and permission changes.

## Compatibility Notes

- Backwards compatibility: existing local/dev Compose flow stays intact; production deploy uses separate artifacts.
- Data/migrations: none; runtime state remains file-backed on the VPS.
- Config/flags: production runtime keeps `config.yaml` external; app TLS remains terminated at Nginx, not the Go server.
- Release/versioning: keep existing release workflow behavior unless explicitly adjusted; deployment workflow is additive or a replacement specifically for image publishing to GHCR.

## API/UX Decisions (if applicable)

- Inputs/outputs: public API remains under the existing app routes served behind `https://api.heweili.top`.
- States/errors: failed deploy should stop on SSH / compose errors rather than silently continuing.
- Telemetry/logging: deployment should make it easy to inspect `docker compose logs` on the VPS after rollout.
- Accessibility/i18n (if UI): not applicable for the first version because the management UI is intentionally not public.
