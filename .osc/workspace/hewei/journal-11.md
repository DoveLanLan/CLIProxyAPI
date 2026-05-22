# Journal 11: Configure CPA-Manager Fork Image Override

- Date: 2026-05-22
- Task: `.osc/tasks/05-22-configurable-cpa-manager-image`

## Decisions

- Keep CPA-Manager source and image publishing in the separate `DoveLanLan/CPA-Manager` fork.
- Keep CLIProxyAPI production deployment responsible only for selecting the CPA-Manager image.
- Use a pinned GHCR SHA image for VPS production instead of the floating `main` tag.

## Changes

- Added `CPA_MANAGER_IMAGE` support in `deploy/compose.production.yml` with default `seakee/cpa-manager:latest`.
- Documented fork image override guidance in `deploy/README.md`.
- Verified the current CPA-Manager fork image:
  - `ghcr.io/dovelanlan/cpa-manager:sha-317e9498bbefa0aa12b5e96837bfd6a7cbc8e3bc`

## Validation

- Default compose render keeps `seakee/cpa-manager:latest`.
- Override compose render uses the pinned GHCR image.
- `git diff --check` passed.
- `go build -o test-output ./cmd/server && rm test-output` passed.

## Next Steps

- Commit and push the CLIProxyAPI deployment change.
- After the production deployment has the updated compose file, set `/opt/cliproxyapi/.env` on the VPS:
  - `CPA_MANAGER_IMAGE=ghcr.io/dovelanlan/cpa-manager:sha-317e9498bbefa0aa12b5e96837bfd6a7cbc8e3bc`
- Restart or redeploy the CPA-Manager service and verify `:18318` still renders monitoring.

## Rollback

- Remove `CPA_MANAGER_IMAGE` from `/opt/cliproxyapi/.env`, or set it to `seakee/cpa-manager:latest`.
- Restart only the CPA-Manager container.
