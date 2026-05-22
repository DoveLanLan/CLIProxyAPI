# Tasks: Fix CPA-Manager Monitoring Load

- Date: 2026-05-22
- Owner(s): hewei
- Related: proposal.md, spec.md

## Assumptions

- The VPS uses `deploy/compose.production.yml` or the same environment shape for CPA-Manager.
- CPA-Manager `USAGE_QUERY_LIMIT` is read on service startup and controls `/v0/management/usage` query size.
- A bounded recent window is acceptable as a pragmatic mitigation until CPA-Manager upstream handles dashboard pagination/timeouts.

## Checklist

- [x] 1) Bound CPA-Manager usage queries
  - Target: `deploy/compose.production.yml`
  - Change: Add `USAGE_QUERY_LIMIT: ${CPA_MANAGER_USAGE_QUERY_LIMIT:-100}` to the `cpa-manager` service environment.
  - Verify: File review and `git diff --name-only`.

- [x] 2) Document VPS remediation
  - Target: `deploy/README.md`
  - Change: Explain the query-limit default, override variable, and lower-limit fallback if the dashboard still times out.
  - Verify: File review.

- [x] 3) Validate and summarize
  - Target: `.osc/tasks/05-22-fix-cpa-manager-monitoring-load/changes/`
  - Change: Update task artifacts with completion notes, regression checklist, and rollback notes.
  - Verify: `git status --short --branch`.

## Notes

- Browser MCP confirmed `/status` succeeds but `/v0/management/usage` and `/v0/management/usage/export` time out.
- CPA-Manager frontend auto-refresh can keep `loading=true` because the refresh interval is shorter than the usage request timeout.
