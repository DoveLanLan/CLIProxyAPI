# Tasks: Docker Compose Rebuild Fix

- Date: 2026-06-08
- Owner(s): Codex
- Related: spec.md, proposal.md

## Assumptions
- Docker Compose service name is `cli-proxy-api`.
- The build should use the repository's existing Docker Compose service and keep runtime configuration bind mounts unchanged.
- Docker's build-time network can be unstable behind the active proxy, so module downloads may need retry and persistent BuildKit cache.

## Checklist
- [x] 1. Add builder-stage Git dependency
  - Target: `Dockerfile`
  - Change: Install `git` before `go mod download`.
  - Verify: `docker compose build cli-proxy-api` passes the dependency download step.

- [x] 2. Stabilize Docker module download and build
  - Target: `Dockerfile`
  - Change: Add BuildKit cache mounts for Go modules/build cache, retry `go mod download`, and disable Go VCS stamping with `-buildvcs=false`.
  - Verify: `docker compose build cli-proxy-api` completes and writes image `cliproxyapi-cli-proxy-api`.

- [x] 3. Rebuild and recreate container
  - Target: Docker Compose service `cli-proxy-api`
  - Change: Build the image and force-recreate the running container.
  - Verify: Container is running and bound to port `8317`.

## Notes
- Initial rebuild failed because `go mod download` could not execute `git version` inside the builder image.
- After Git was added, several builds failed because Docker build networking returned `unexpected EOF` while downloading Go modules from GitHub/Go proxies.
- The final build succeeded with image ID `sha256:48f93f77e84a280f908c1aeed58d214057ffccf8fc29b618cef2eb94d8f3be81`.
- `docker compose up -d --force-recreate cli-proxy-api` recreated and started container `cli-proxy-api` on port `8317`.
