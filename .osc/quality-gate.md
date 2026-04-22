# Quality Gate Report

- Date: 2026-04-22
- Task: `.osc/tasks/04-21-shared-vps-gateway`

**Assumptions:**
- This change is deployment-only and does not alter Go server behavior.
- Shared public nginx ownership is now `/opt/vps-gateway` with container `vps-gateway-nginx`.
- The repo's minimum CI gate remains the compile step in `.github/workflows/pr-test-build.yml`.

**Suspected Change Scope:**
- `deploy/compose.production.yml`
- `deploy/.env.example`
- `deploy/compose.production.split-proxy.yml`
- `deploy/scripts/remote-deploy.sh`
- `deploy/nginx/conf.d/api.heweili.top.conf`
- `deploy/README.md`
- `deploy/SPLIT_PROXY_SETUP_CN.md`
- `deploy/split-proxy/start.sh`
- `deploy/split-proxy/README.md`
- `docker-compose.split-proxy.yml`
- `.osc/tasks/04-21-shared-vps-gateway/changes/*`

**Detected Gates:**
- Gate Name: PR build compile check
  - Confidence: High
  - Evidence: `.github/workflows/pr-test-build.yml` runs `go build -o test-output ./cmd/server`
- Gate Name: Production compose rendering
  - Confidence: High
  - Evidence: `deploy/compose.production.yml`, `deploy/scripts/remote-deploy.sh`
- Gate Name: Split-proxy compose rendering
  - Confidence: High
  - Evidence: `deploy/compose.production.split-proxy.yml`, `deploy/scripts/remote-deploy.sh`
- Gate Name: Deployment shell syntax validation
  - Confidence: High
  - Evidence: `deploy/scripts/remote-deploy.sh`
- Gate Name: Gateway nginx validation
  - Confidence: Medium
  - Evidence: `deploy/scripts/remote-deploy.sh` runs `docker exec "$GATEWAY_CONTAINER" nginx -t`
- Gate Name: Split-proxy local upstream network validation
  - Confidence: High
  - Evidence: `deploy/compose.production.split-proxy.yml`, `deploy/scripts/remote-deploy.sh`

**Executed Gates:**
- `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml config`
  - Result: passed
- `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml config`
  - Result: passed; rendered `split-proxy` on both `proxy` / `vps-gateway` and `local-claude` / `cli-proxy-api-proxy`.
- `UPSTREAM_PROXY_HOST=proxy.example.com UPSTREAM_PROXY_PORT=3128 docker compose -f docker-compose.yml -f docker-compose.split-proxy.yml config`
  - Result: passed
- `bash -n deploy/scripts/remote-deploy.sh`
  - Result: passed
- `go build -o test-output ./cmd/server`
  - Result: passed; generated `test-output` was removed after verification.

**Previously Executed Remote Gates:**
- 2026-04-21 remote sync and deploy on `bytevirt`: uploaded deploy assets to `/opt/cliproxyapi` and ran `CLI_PROXY_IMAGE=ghcr.io/dovelanlan/cliproxyapi:main bash scripts/remote-deploy.sh`.
  - Result: passed
- 2026-04-21 remote gateway validation: `docker exec vps-gateway-nginx nginx -t`.
  - Result: passed
- 2026-04-21 remote public route sanity: `curl -I http://23.175.201.12 -H "Host: api.heweili.top"`.
  - Result: passed with HTTP `301` to `https://api.heweili.top/`
- 2026-04-22 remote gateway validation after split-proxy network fix: `docker exec vps-gateway-nginx nginx -t`.
  - Result: passed; nginx reported configuration syntax is OK and test is successful.
- 2026-04-22 remote split-proxy DNS validation after split-proxy network fix: `docker exec cli-proxy-split-proxy getent hosts kiro-rs`.
  - Result: passed; `kiro-rs` resolved to `172.18.0.4`.

**Final Self-Review:**
- Security & secrets: no live credentials were committed; runtime settings remain in server `.env`.
- Edge cases & error handling: deploy script fails clearly if gateway config dir or gateway container is missing.
- Edge cases & error handling: deploy script fails clearly if split-proxy is enabled and `LOCAL_CLAUDE_NETWORK` is missing.
- Backward compatibility / migrations: no app data or API behavior changes.
- API/contract compatibility: public API and management route policy remain unchanged.
- Observability: use GitHub Actions logs, Docker Compose output, nginx logs, and container logs.
- Config/env changes: new optional `GATEWAY_NETWORK`, `GATEWAY_ROOT`, and `GATEWAY_CONTAINER` values are documented.
- Config/env changes: `LOCAL_CLAUDE_NETWORK` is documented and defaults to `cli-proxy-api-proxy` for production split-proxy.
- Performance risk: no runtime hot path change.
- Rollback plan: documented in task rollback notes.

**PR-ready checklist:**
- [x] `go build -o test-output ./cmd/server`
- [x] `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml config`
- [x] `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml config`
- [x] `UPSTREAM_PROXY_HOST=proxy.example.com UPSTREAM_PROXY_PORT=3128 docker compose -f docker-compose.yml -f docker-compose.split-proxy.yml config`
- [x] `bash -n deploy/scripts/remote-deploy.sh`
- [x] remote `docker exec vps-gateway-nginx nginx -t` after the 2026-04-22 network fix is deployed
- [x] remote `docker exec cli-proxy-split-proxy getent hosts kiro-rs` after the 2026-04-22 network fix is deployed
- [x] GitHub Actions production workflow run after commit/push
