# Rollback Notes: Fix CPA-Manager Monitoring Load

- Date: 2026-05-22
- Related: spec.md, tasks.md

## Rollback

- Remove `USAGE_QUERY_LIMIT` from `deploy/compose.production.yml`, or set `CPA_MANAGER_USAGE_QUERY_LIMIT=50000` in `/opt/cliproxyapi/.env`.
- Restart only the CPA-Manager container after changing the environment.

## Data Considerations

- No SQLite schema or data migration is introduced.
- Existing `/opt/cliproxyapi/data/cpa-manager/usage.sqlite` data is preserved.

## Follow-up Remediation

- If bounded queries still hang even at `100`, inspect the CPA-Manager SQLite database for unusually large `raw_json` rows or rotate/prune the CPA-Manager data volume manually after backing it up.
- Upstream CPA-Manager should fix the dashboard endpoint to avoid selecting raw event payloads when building normal monitoring summaries.
