# Change Summary: Docker Compose Rebuild Fix

- Date: 2026-06-08
- Owner(s): Codex
- Related: spec.md, tasks.md

## What changed
- Updated the Docker builder stage to install `git`, which is required when Go module resolution falls back to VCS access.
- Added BuildKit cache mounts for `/go/pkg/mod` and `/root/.cache/go-build`, plus retry logic around `go mod download`, so interrupted proxy downloads do not force every rebuild to start from zero.
- Added `-buildvcs=false` to the container build command because Docker build context does not provide reliable VCS metadata for Go stamping; explicit `VERSION`, `COMMIT`, and `BUILD_DATE` ldflags remain in place.
- Rebuilt image `cliproxyapi-cli-proxy-api` and force-recreated container `cli-proxy-api`.

## Why
`docker compose build cli-proxy-api` initially failed during Go module download because the builder image did not include `git`. After that was fixed, repeated Docker build attempts failed because the build container received `unexpected EOF` from GitHub/Go module proxies. The Dockerfile now tolerates transient module download failures and can reuse completed downloads across attempts.

## Notable decisions
- Keep `git`, module caches, and retry logic in the builder stage only; the runtime image remains Alpine plus the compiled binary and config template.
- Disable Go VCS stamping in Docker builds and rely on existing ldflags for version metadata.
- Runtime bind mounts for `config.yaml`, `auths`, and `logs` were left unchanged.
