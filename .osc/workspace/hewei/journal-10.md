# hewei journal 10

- Date: 2026-05-22
- Title: Fix CPA-Manager monitoring load
- Task: `.osc/tasks/05-22-fix-cpa-manager-monitoring-load`

## Summary

Conclusions/decisions: Browser MCP showed the VPS CPA-Manager panel can reach metadata endpoints, but `/v0/management/usage` and `/v0/management/usage/export` do not return within the frontend timeout. The UI auto-refresh interval is shorter than the usage request timeout, so repeated slow requests keep the loading overlay visible. The immediate repo-side mitigation is to bound CPA-Manager's dashboard query window with `USAGE_QUERY_LIMIT`.

What changed: Added `USAGE_QUERY_LIMIT: ${CPA_MANAGER_USAGE_QUERY_LIMIT:-1000}` to `deploy/compose.production.yml` and documented the override/lower-limit fallback in `deploy/README.md`.

Verification: `docker compose config` rendered successfully with `USAGE_QUERY_LIMIT: "1000"`; `git diff --check` passed; `go build -o test-output ./cmd/server && rm test-output` passed.

Risks/rollback: This is a deployment mitigation, not an upstream CPA-Manager fix. It limits monitoring history shown by the dashboard. Roll back by removing the env var or setting `CPA_MANAGER_USAGE_QUERY_LIMIT=50000`, then restart only `cpa-manager`.

Next steps: Deploy the compose change to the VPS and restart `cpa-manager`. If the page still times out, lower `CPA_MANAGER_USAGE_QUERY_LIMIT` to `200` or `100` and inspect CPA-Manager SQLite `raw_json` row sizes.
