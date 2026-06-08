# Journal 13: Docker Compose Rebuild Fix

- Date: 2026-06-08
- Related task: `.osc/tasks/06-08-docker-compose-rebuild-fix/changes/`

## Conclusion

The local Docker Compose image was rebuilt from the current code and the `cli-proxy-api` container was force-recreated successfully.

The original build failed because the `golang:1.26-alpine` builder image did not include `git`, and later retries failed due to transient Go module download EOFs through Docker build networking. The Dockerfile now installs `git` in the builder stage, uses BuildKit cache mounts for Go module/build cache, retries `go mod download`, and disables Go's automatic VCS stamping with `-buildvcs=false` while keeping explicit ldflags for `VERSION`, `COMMIT`, and `BUILD_DATE`.

## Verification

- `GOPROXY='https://proxy.golang.org,https://goproxy.cn,https://goproxy.io,direct' GOSUMDB='sum.golang.org' docker compose build cli-proxy-api` passed.
- Built image: `sha256:48f93f77e84a280f908c1aeed58d214057ffccf8fc29b618cef2eb94d8f3be81`.
- `docker compose up -d --force-recreate cli-proxy-api` passed.
- `docker ps --filter name=cli-proxy-api` showed the container running with port `8317` mapped.
- `docker logs --tail 100 cli-proxy-api` showed successful startup and management route responses; unrelated auth refresh warnings and upstream/API 502 entries remain.

## Next Step

Manually retest `http://localhost:8317/management.html#/config` and check whether saving `config.yaml` still reports `notification.save_failed: invalid_yaml`.

## Risks

- BuildKit is now required for this Dockerfile because it uses `RUN --mount=type=cache`.
- Docker builder cache can consume additional Docker cache storage.
- `-buildvcs=false` removes Go automatic VCS build info, but project version metadata remains injected via ldflags.

## Rollback

Revert the Dockerfile changes from this task and rebuild/recreate the compose service, or redeploy a previously known-good image.
