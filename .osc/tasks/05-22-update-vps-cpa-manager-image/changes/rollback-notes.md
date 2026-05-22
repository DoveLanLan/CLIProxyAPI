# Rollback Notes: Update VPS CPA-Manager Image

- Date: 2026-05-22
- Related: spec.md, tasks.md

## Rollback

- Restore the timestamped `.env.bak.*` file created on the VPS, or set:
  - `CPA_MANAGER_IMAGE=seakee/cpa-manager:latest`
- Restart only CPA-Manager:
  - `docker compose -f /opt/cliproxyapi/compose.production.yml up -d cpa-manager`

## Data Considerations

- No CPA-Manager SQLite data or volume changes are introduced.
