# Rollback Notes: Docker Compose Rebuild Fix

- Date: 2026-06-08
- Related: spec.md, tasks.md

## Rollback
- Revert the Dockerfile changes from this task: the Dockerfile syntax directive, builder-stage `git` install, BuildKit cache mounts, `go mod download` retry loop, and `-buildvcs=false` flag.
- Rebuild and recreate the container with Docker Compose.
- If rollback is needed quickly, retag or redeploy the previously known-good image before rebuilding.

## Data/config impact
- No data or config migration is involved.
- Existing bind mounts for `config.yaml`, `auths`, and `logs` are unchanged; rollback does not require data movement.
