# Tasks: Set Up GHCR GitHub Actions Deployment To HK VPS

- Date: 2026-03-26
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`

## Assumptions

- The GitHub repository remains public during the first deployment iteration.
- The deployment target is `root@23.175.201.12:22`.
- Cloudflare will proxy `api.heweili.top` and use `Full (strict)` with an Origin CA certificate installed on the VPS.
- Production `config.yaml`, `auths/`, `logs/`, and TLS certificate files will be provisioned manually on the VPS and not committed to git.

## Checklist

- [x] 1) Add production deployment artifacts
  - Target: new `deploy/` directory, production Compose file, Nginx config, env/example docs
  - Change: create a dedicated production stack for `api.heweili.top` using Nginx as the only public entrypoint and the app container as an internal service
  - Verify: inspect files for public `80/443` only, internal app wiring, and management-path blocking

- [x] 2) Rework image publishing for the fork
  - Target: `.github/workflows/docker-image.yml` or replacement workflow(s)
  - Change: publish `linux/amd64` images to GHCR under the fork namespace with traceable tags and repo metadata
  - Verify: workflow syntax is valid and image tags resolve to GHCR, not Docker Hub

- [x] 3) Add production deploy workflow
  - Target: new deploy workflow plus any helper script/docs needed
  - Change: add SSH-based deployment that prepares the remote directory, synchronizes deployment files, pulls the latest image, and recreates the stack
  - Verify: workflow steps include strict SSH setup, remote `mkdir -p`, `docker compose pull`, and `docker compose up -d`

- [x] 4) Document Cloudflare and server prerequisites
  - Target: deployment docs / example env file / final notes
  - Change: capture the required Cloudflare dashboard settings, origin certificate placement, GHCR package visibility expectation, and server-side persistent files
  - Verify: the docs let the operator complete initial server bootstrap without guessing hidden prerequisites

- [x] 5) Run repo quality checks for workflow and deployment changes
  - Target: updated workflows and deployment artifacts
  - Change: run the relevant build/validation gates and capture results in `.osc/quality-gate.md`
  - Verify: at minimum `go build ./cmd/server` still passes and deployment file syntax is checked where practical

## Notes

- The current root `docker-compose.yml` is intentionally local/dev-oriented and should not be overloaded with production-specific proxy/TLS concerns.
- The first version will keep management access private; a later follow-up can use Tailscale or SSH tunneling for admin-only access.
- GHCR image publication was simplified to amd64-only for the current VPS target.
