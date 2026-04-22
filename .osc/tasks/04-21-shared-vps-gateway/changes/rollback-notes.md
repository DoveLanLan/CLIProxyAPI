# Rollback Notes: Move CLIProxyAPI Deploy to Shared VPS Gateway

- Date: 2026-04-21
- Related: spec.md, tasks.md

## Rollback strategy

To roll back repo changes, revert this task's changes in `deploy/`. To roll back the VPS runtime migration, restore the backed-up remote compose file:

```bash
cp /opt/cliproxyapi/compose.production.yml.bak.20260421-063148 /opt/cliproxyapi/compose.production.yml
cd /opt/cliproxyapi
docker compose -f compose.production.yml -f compose.production.split-proxy.yml up -d --remove-orphans
```

If only the 2026-04-22 split-proxy network change needs rollback, revert the `local-claude` network block from `deploy/compose.production.split-proxy.yml` and remove `LOCAL_CLAUDE_NETWORK` from the server `.env`. If the running container still has the manually added network attachment, detach it with:

```bash
docker network disconnect cli-proxy-api-proxy cli-proxy-split-proxy
```

Then stop the shared gateway if desired:

```bash
cd /opt/vps-gateway
docker compose -f compose.yml down
```

## Data / migration considerations

- No application data or config data was migrated.
- `/opt/cliproxyapi/data/*` and auth files are unchanged.
- Existing certificates remain under `/opt/cliproxyapi/certs` and are mounted by the shared gateway.
- The split-proxy network fix only changes Docker network attachment; it does not change persisted auth/config data.

## Operational notes

- Watch `docker ps`, `docker logs vps-gateway-nginx`, and `docker logs cli-proxy-api`.
- Watch `/opt/cliproxyapi/data/logs/split-proxy/access.log` for `TCP_MISS/502` entries to `http://kiro-rs:8990`.
- Public API should continue returning HTTP-to-HTTPS redirect on port 80 and API responses on 443 through Cloudflare.
- Known residual effect: the gateway stack is now a shared dependency for CLIProxyAPI public access.
