# Regression Checklist: Set Up GHCR GitHub Actions Deployment To HK VPS

- Date: 2026-03-26
- Related: `spec.md`, `tasks.md`

## Gates (from Repo Snapshot)

- Build: `go build -o /tmp/cli-proxy-api-check ./cmd/server`
- Workflow/config syntax: Ruby YAML parse over `.github/workflows/*.yml` and `deploy/compose.production.yml`
- Compose validation: `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml config`
- Shell validation: `bash -n deploy/scripts/remote-deploy.sh`
- Nginx validation: `docker run ... nginx:1.27-alpine nginx -t` on a temporary Docker network with a `cli-proxy-api` alias

## Manual checks (if applicable)

- Confirm the first GHCR package publish appears under the fork and set its visibility to `public` if anonymous VPS pulls are desired.
- Confirm Cloudflare `api.heweili.top` DNS record is proxied and `SSL/TLS` mode is `Full (strict)`.
- Confirm `/opt/cliproxyapi/data/config.yaml`, `/opt/cliproxyapi/certs/origin.crt`, and `/opt/cliproxyapi/certs/origin.key` exist before the first deploy run.
- Confirm `https://api.heweili.top/management.html` is not public after deployment.

## Edge-case re-tests

- Re-run the deploy workflow with the same image tag to confirm the remote deployment is idempotent.
- Run a CLI streaming request through `https://api.heweili.top` and confirm long-lived responses are not buffered or cut off.
- Confirm `/v1/ws` still upgrades correctly if WebSocket clients are used.
