# Regression Checklist: Docker Compose Rebuild Fix

- Date: 2026-06-08
- Related: spec.md, tasks.md

## Gates (from Repo Snapshot)
- Build: `GOPROXY='https://proxy.golang.org,https://goproxy.cn,https://goproxy.io,direct' GOSUMDB='sum.golang.org' docker compose build cli-proxy-api` (passed; image `sha256:48f93f77e84a280f908c1aeed58d214057ffccf8fc29b618cef2eb94d8f3be81`)
- Runtime: `docker compose up -d --force-recreate cli-proxy-api` (passed; container started)
- Container status: `docker ps --filter name=cli-proxy-api` (passed; port `8317` exposed)
- Startup logs: `docker logs --tail 100 cli-proxy-api` (passed for startup; unrelated upstream/auth warnings remain)

## Manual Checks
- Confirm `cli-proxy-api` is running. Done.
- Open `http://localhost:8317/management.html#/config` and retry saving config. Manual user check still required.

## Observed Non-blocking Log Items
- Startup succeeded and management routes responded with HTTP 200.
- Logs contain unrelated auth refresh warnings (`refresh_token_reused`) and a management API call 502; these were not part of the Docker rebuild request.
