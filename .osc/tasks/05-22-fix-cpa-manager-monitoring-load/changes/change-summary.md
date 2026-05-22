# Change Summary: Fix CPA-Manager Monitoring Load

- Date: 2026-05-22
- Owner(s): hewei
- Related: proposal.md, spec.md, tasks.md

## What changed

- Added `USAGE_QUERY_LIMIT: ${CPA_MANAGER_USAGE_QUERY_LIMIT:-100}` to the production CPA-Manager service environment.
- Documented `CPA_MANAGER_USAGE_QUERY_LIMIT` in the VPS deployment guide, including raising it only when a larger historical window returns before the frontend timeout.
- Recorded browser findings: `/status` succeeds, while `/v0/management/usage` and `/v0/management/usage/export` time out; frontend auto-refresh can keep the loading overlay active.

## Why

CPA-Manager's dashboard currently asks its Usage Service for a large recent-event window on each refresh. On the VPS, that endpoint does not return before the frontend timeout, and auto-refresh starts new usage requests before older requests can settle. Bounding the query window is a reversible deployment-side mitigation that should let the page load recent monitoring data while preserving the existing service topology.

## Notable decisions

- Keep this repository free of CPA-Manager source vendoring.
- Do not delete or mutate existing SQLite usage data automatically.
- Leave the upstream CPA-Manager fixes as follow-up work: dashboard pagination/timeout handling and avoiding unnecessary raw event reads for normal monitoring payloads.
