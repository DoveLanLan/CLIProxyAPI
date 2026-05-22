# Tasks: Make CPA-Manager Image Configurable

- Date: 2026-05-22
- Owner(s): hewei
- Related: proposal.md, spec.md

## Assumptions

- The VPS uses `deploy/compose.production.yml` or the same environment shape.
- The CPA-Manager fork will publish an image compatible with the current `18317` container port and `/data` storage.

## Checklist

- [x] 1) Parameterize production image
  - Target: `deploy/compose.production.yml`
  - Change: Replace the hardcoded CPA-Manager image with `${CPA_MANAGER_IMAGE:-seakee/cpa-manager:latest}`.
  - Verify: Compose render shows the default image when unset and the forked image when set.

- [x] 2) Document fork image override
  - Target: `deploy/README.md`
  - Change: Add `CPA_MANAGER_IMAGE` to the server bootstrap variables and note fixed-tag fork deployment guidance.
  - Verify: File review.

- [x] 3) Validate deployment config
  - Target: deployment config and repo hygiene
  - Change: Run compose config rendering and diff/build checks.
  - Verify: `docker compose config`, `git diff --check`, and `go build -o test-output ./cmd/server && rm test-output`.

## Notes

- Keep `USAGE_QUERY_LIMIT` capped at `100` until CPA-Manager fork fixes query performance and overlapping frontend polling.
