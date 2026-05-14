# Change Summary: Integrate CPA-Manager Panel and Usage Monitoring

- Date: 2026-05-14
- Owner(s): hewei
- Related: proposal.md, spec.md, tasks.md

## What changed

- Added an in-memory usage queue and `GET /v0/management/usage-queue` for CPA-Manager HTTP collection.
- Added `GET /v0/management/api-key-usage` and auth recent-request counters.
- Extended usage records with provider/model/alias/auth/token/latency/failure metadata.
- Switched the default management panel repository and fallback download to `seakee/CPA-Manager`.
- Added compose and production deployment examples for the external CPA-Manager Usage Service.

## Why

CPA-Manager can replace the current `/management.html` panel directly, but its request monitoring requires a Usage Service and a queue endpoint that this branch did not have. The implementation adds only the compatibility surface needed for that workflow.

## Notable decisions

- Did not vendor CPA-Manager source into this repository.
- Did not upgrade the entire codebase to upstream v6.10.8 or v7.
- Did not implement RESP queue compatibility; HTTP usage queue is the supported integration path.
