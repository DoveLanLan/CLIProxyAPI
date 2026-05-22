# Rollback Notes: Make CPA-Manager Image Configurable

- Date: 2026-05-22
- Related: spec.md, tasks.md

## Rollback

- Remove `CPA_MANAGER_IMAGE` from `/opt/cliproxyapi/.env`, or set it back to `seakee/cpa-manager:latest`.
- Restart only the CPA-Manager container after changing the image value.

## Data Considerations

- No SQLite schema, data, or volume changes are introduced.
- Existing `/opt/cliproxyapi/data/cpa-manager/usage.sqlite` data is preserved.

## Follow-up Remediation

- CPA-Manager fork fixes should still address backend usage query performance and frontend overlapping polling before increasing `CPA_MANAGER_USAGE_QUERY_LIMIT`.
