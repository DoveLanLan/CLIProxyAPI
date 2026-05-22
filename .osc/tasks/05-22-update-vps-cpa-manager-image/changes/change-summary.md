# Change Summary: Update VPS CPA-Manager Image

- Date: 2026-05-22
- Owner(s): hewei
- Related: proposal.md, spec.md, tasks.md

## What changed

- Added `.github/workflows/update-cpa-manager-image.yml`.
- The workflow updates only `CPA_MANAGER_IMAGE` in `/opt/cliproxyapi/.env` and restarts `cpa-manager`.

## Why

Local SSH access is not available, while GitHub Actions already owns production SSH deployment credentials.

## Notable decisions

- Validate images to the `ghcr.io/dovelanlan/cpa-manager:*` namespace.
- Keep a timestamped backup of the VPS `.env` before editing.
