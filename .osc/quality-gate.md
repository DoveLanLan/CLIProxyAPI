# Quality Gate Report

- Date: 2026-04-09
- Task: `.osc/tasks/04-09-fix-split-proxy-squid-logging-startup`

**Assumptions:**
- The split-proxy startup failure is isolated to deploy-side Squid configuration and not to the main Go server runtime.
- The production server continues to use the deploy assets under `/opt/cliproxyapi/` uploaded from `deploy/`.
- The repo's minimum CI gate remains the compile step in `.github/workflows/pr-test-build.yml`.

**Suspected Change Scope:**
- `deploy/split-proxy/start.sh`
- `deploy/compose.production.split-proxy.yml`
- `docker-compose.split-proxy.yml`
- `deploy/SPLIT_PROXY_SETUP_CN.md`
- `deploy/split-proxy/README.md`
- `deploy/README.md`

**Detected Gates:**
- Gate Name: PR build compile check
  - Confidence: High
  - Evidence: `.github/workflows/pr-test-build.yml` runs `go build -o test-output ./cmd/server`
- Gate Name: Split-proxy shell syntax validation
  - Confidence: High
  - Evidence: `deploy/split-proxy/start.sh` is the deployed entrypoint script for the Squid sidecar
- Gate Name: Production compose rendering
  - Confidence: High
  - Evidence: `deploy/scripts/remote-deploy.sh` conditionally includes `deploy/compose.production.split-proxy.yml`
- Gate Name: Local compose rendering
  - Confidence: Medium
  - Evidence: `docker-compose.split-proxy.yml` is the repo-local split-proxy override for local/docker testing
- Gate Name: Documentation/operator guidance review
  - Confidence: Medium
  - Evidence: `deploy/SPLIT_PROXY_SETUP_CN.md`, `deploy/split-proxy/README.md`, `deploy/README.md` define the operator workflow for this feature

**Executed Gates (Local):**
- `bash -n deploy/split-proxy/start.sh`
  - Result: passed
- `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml config >/tmp/cliproxy-prod-split-proxy-config.yaml`
  - Result: passed
- `UPSTREAM_PROXY_HOST=proxy.example.com UPSTREAM_PROXY_PORT=3128 docker compose -f docker-compose.yml -f docker-compose.split-proxy.yml config >/tmp/cliproxy-local-split-proxy-config.yaml`
  - Result: passed
- `go build -o /tmp/cli-proxy-api-check ./cmd/server`
  - Result: passed

**Final Self-Review:**
- Security & secrets: no live proxy credentials or server secrets were committed; docs continue to direct operators to server-side `.env`.
- Edge cases & error handling: the startup script now prepares/chowns both the Squid log and spool directories before launch.
- Backward compatibility / migrations: existing split-proxy env vars and `proxy-url` usage remain unchanged; only log persistence behavior changes.
- API/contract compatibility: no public API, SDK, or management contract changed.
- Observability: split-proxy logs move to persisted files under the mounted log directory; docs were updated to reflect the new inspection path.
- Config/env changes: no new env vars were introduced; new runtime expectation is the mounted `split-proxy` log directory under existing host logs.
- Performance risk: low; writing Squid logs to regular files is a normal path and does not alter proxy routing.
- Rollback plan: revert the deploy asset changes and redeploy; note that the original stdio logging crash would return on affected hosts.

**PR-ready checklist:**
- [x] `bash -n deploy/split-proxy/start.sh`
- [x] `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml config`
- [x] `UPSTREAM_PROXY_HOST=proxy.example.com UPSTREAM_PROXY_PORT=3128 docker compose -f docker-compose.yml -f docker-compose.split-proxy.yml config`
- [x] `go build -o /tmp/cli-proxy-api-check ./cmd/server`
- [ ] Live redeploy to a server with `ENABLE_SPLIT_PROXY=true`
  - Confirm `cli-proxy-split-proxy` stays up and that `data/logs/split-proxy/access.log` plus `cache.log` are created on the server.
