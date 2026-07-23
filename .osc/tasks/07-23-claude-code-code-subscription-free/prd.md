# Bugfix: xAI streamed free-usage exhaustion

## Problem

Claude Code frequently receives `subscription:free-usage-exhausted` from
`grok-4.5-build-free` even while hundreds of xAI credentials remain active.
Production request logs record HTTP 200 because the quota failure is embedded
inside the HTTP SSE stream instead of being returned as the initial HTTP status.

The production inspection job also permanently disables quota-exhausted free
accounts and never re-enables disabled accounts, although their quota recovers
over a rolling 24-hour window.

## Reproduction

1. Configure multiple xAI free credentials and use the HTTP SSE executor.
2. Have the upstream return HTTP 200 followed by an SSE data payload containing
   `subscription:free-usage-exhausted`.
3. Observe that the payload reaches Claude Code instead of triggering auth
   cooldown and credential failover.

## Expected behavior

- A streamed free-usage error is converted into a 429 execution error before
  any downstream payload is committed.
- The exhausted credential enters a 24-hour model cooldown and the request is
  retried with another eligible credential.
- Rolling-quota credentials are recoverable; only permanently invalid
  credentials are deleted after a fresh inspection and backup.

## Actual behavior

- HTTP SSE error payloads are translated as normal stream data.
- Claude Code sees a stream-level 429 while Gin records HTTP 200.
- The inspection job monotonically shrinks the pool by permanently disabling
  quota-exhausted credentials.

## Root cause

- `XAIExecutor.ExecuteStream` checks non-2xx HTTP responses but does not classify
  xAI error objects received inside a successful SSE response.
- The production inspection script includes `quota_exhausted` in its permanent
  disable classes, sets `include_disabled=0`, and explicitly never enables an
  account.
- Cooldown persistence is not enabled in production.

## Fix

- Detect streamed xAI free-usage errors before protocol translation and emit a
  `StreamChunk.Err` carrying status 429 and the existing 24-hour retry hint.
- Add focused executor and auth-manager failover regression tests.
- Enable cooldown persistence and use round-robin production routing.
- Replace permanent quota disablement with recoverable quarantine semantics.
- Reinspect disabled credentials; back up and delete only hard-dead classes.

## Regression tests

- [ ] Streamed free-usage payload becomes a 429 error with a 24-hour retry hint.
- [ ] The auth manager cools the first credential and succeeds with the next.
- [ ] Normal xAI SSE events remain unchanged.
- [ ] Required focused tests and server build pass.
- [ ] Production smoke test completes without exposing credentials.
