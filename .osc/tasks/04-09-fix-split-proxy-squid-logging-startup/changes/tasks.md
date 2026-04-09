# Tasks: Fix split-proxy Squid logging startup failure

- Date: 2026-04-09
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`

## Assumptions

- The current failure is caused by Squid trying to open `/dev/stdout` and `/dev/stderr` after dropping privileges to the `proxy` user.
- The deploy workflows continue to ship only files under `deploy/`, so fixing the deployed startup script and docs is sufficient.
- Compose validation and shell syntax checks are the most relevant local gates for this change.

## Checklist

- [x] 1) Update split-proxy startup script logging
  - Target: `deploy/split-proxy/start.sh`
  - Change: write Squid logs to writable files under `/var/log/squid`, prepare runtime directories, preserve existing routing config
  - Verify: `bash -n deploy/split-proxy/start.sh`

- [x] 2) Persist split-proxy log directory through compose
  - Target: `deploy/compose.production.split-proxy.yml`, `docker-compose.split-proxy.yml`
  - Change: mount a host split-proxy log directory into `/var/log/squid`
  - Verify: `docker compose ... config` renders the updated volume mounts successfully

- [x] 3) Update operator docs
  - Target: `deploy/SPLIT_PROXY_SETUP_CN.md`, `deploy/split-proxy/README.md`
  - Change: document the new log location, inspection commands, and startup-failure recovery note
  - Verify: manual review of the updated setup steps and troubleshooting section

- [x] 4) Run targeted deploy-file validation
  - Target: `deploy/`, `.osc/quality-gate.md`
  - Change: run shell/compose/build checks relevant to deploy assets and record the gate results
  - Verify: updated `.osc/quality-gate.md`

## Notes

- Keep the fix minimal and deploy-focused; do not expand into unrelated proxy behavior changes.
- Validation completed:
  - `bash -n deploy/split-proxy/start.sh`
  - `docker compose --env-file deploy/.env.example -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml config`
  - `UPSTREAM_PROXY_HOST=proxy.example.com UPSTREAM_PROXY_PORT=3128 docker compose -f docker-compose.yml -f docker-compose.split-proxy.yml config`
  - `go build -o /tmp/cli-proxy-api-check ./cmd/server`
