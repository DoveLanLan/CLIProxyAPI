# Rollback Notes: Fix split-proxy Squid logging startup failure

- Date: 2026-04-09
- Related: `proposal.md`, `spec.md`, `tasks.md`

## Rollback strategy

Revert the deploy asset changes in:

- `deploy/split-proxy/start.sh`
- `deploy/compose.production.split-proxy.yml`
- `docker-compose.split-proxy.yml`
- `deploy/SPLIT_PROXY_SETUP_CN.md`
- `deploy/split-proxy/README.md`
- `deploy/README.md`

Then redeploy the `deploy/` directory to the server and rerun `bash scripts/remote-deploy.sh`.

## Data / migration considerations

- No schema or persistent app data migration is involved.
- The only new runtime artifact is the host log directory `data/logs/split-proxy/`.

## Operational notes

- Monitoring/alerts to watch: the `cli-proxy-split-proxy` container status and the creation of `access.log` / `cache.log`.
- Known residual effects: after rollback, the original `/dev/stdout` / `/dev/stderr` startup failure will return on hosts that use the same Squid image/runtime behavior.
