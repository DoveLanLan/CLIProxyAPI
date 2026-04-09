# hewei journal 2

- Date: 2026-04-09
- Title: Fix split-proxy Squid logging startup failure
- Commit: 

## Summary
Root cause: Squid sidecar crashed because access_log/cache_log targeted /dev/stdout and /dev/stderr while running as proxy user. Fix: write logs to /var/log/squid/*.log, prepare/chown log+spool dirs, mount persistent split-proxy log dirs in compose, and update deploy docs. Validation: bash -n deploy/split-proxy/start.sh; docker compose config for production/local split-proxy overrides; go build ./cmd/server. Next: redeploy on server and confirm cli-proxy-split-proxy stays Up and creates access.log/cache.log. Rollback: revert deploy-side split-proxy files and redeploy; old stdio crash would return.
