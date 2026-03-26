# Quality Gate Report

- Date: 2026-03-26
- Task: `.osc/tasks/03-26-ghcr-vps-nginx-deploy`

**Assumptions:**
- The production runtime secrets remain on the VPS and are intentionally absent from this repository.
- The first production rollout uses Cloudflare Origin CA, not Let's Encrypt.
- The deployment workflows are designed for the current fork and current VPS target only.

**Suspected Change Scope:**
- `.github/workflows/docker-image.yml`
- `.github/workflows/deploy-production.yml`
- `deploy/`
- `.osc/tasks/03-26-ghcr-vps-nginx-deploy/changes/`

**Detected Gates:**
- Gate Name: PR build compile check
  - Confidence: High
  - Evidence: `.github/workflows/pr-test-build.yml` runs `go build -o test-output ./cmd/server`
- Gate Name: Workflow syntax sanity
  - Confidence: Medium
  - Evidence: tracked changes add GitHub Actions YAML that must remain parseable
- Gate Name: Production Compose validation
  - Confidence: High
  - Evidence: `deploy/compose.production.yml` is the new runtime entrypoint for VPS deployment
- Gate Name: Nginx reverse proxy validation
  - Confidence: High
  - Evidence: `deploy/nginx/conf.d/api.heweili.top.conf` defines the new public ingress path
- Gate Name: Shell deploy script validation
  - Confidence: High
  - Evidence: `deploy/scripts/remote-deploy.sh` is executed remotely by the new workflow

**Executed Gates (Local):**
- `ruby -e 'require "yaml"; ...'`
  - Result: passed for `.github/workflows/docker-image.yml`, `.github/workflows/deploy-production.yml`, and `deploy/compose.production.yml`
- `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml config`
  - Result: passed
- `bash -n deploy/scripts/remote-deploy.sh`
  - Result: passed
- `go build -o /tmp/cli-proxy-api-check ./cmd/server`
  - Result: passed
- `docker run ... nginx:1.27-alpine nginx -t` on a temporary network with a `cli-proxy-api` alias
  - Result: passed

**Final Self-Review:**
- Security & secrets: no live production secrets were committed; workflows expect SSH material through GitHub environment secrets.
- Edge cases & error handling: remote deploy fails fast if `config.yaml` or Origin CA files are missing on the VPS.
- Backward compatibility / migrations: root local/dev Compose flow remains intact; production deploy is isolated under `deploy/`.
- API/contract compatibility: public API paths stay unchanged; only the ingress and delivery mechanism are added.
- Config/env changes: new production stack expects `CLI_PROXY_IMAGE` and server-side runtime files under `/opt/cliproxyapi`.
- Config/env changes: new production stack expects `PUBLIC_BIND_IP`, `TAILSCALE_BIND_IP`, and `TAILSCALE_MANAGEMENT_PORT` so public ingress and private management can coexist with Tailscale Serve on `443`.
- Performance risk: low; Nginx is configured for streaming-friendly proxy behavior and WebSocket upgrades.
- Rollback plan: redeploy an older GHCR image tag or revert the workflow / `deploy/` changes.

**PR-ready checklist:**
- [x] GitHub workflow YAML parses
- [x] Production Compose expands successfully
- [x] Remote deploy shell script passes `bash -n`
- [x] Nginx config passes `nginx -t` in a simulated Docker network
- [x] `go build ./cmd/server` still passes
- [ ] Live GitHub Actions deploy to VPS
  - Pending real environment secrets and the first server-side bootstrap (`config.yaml` + Origin CA files).
