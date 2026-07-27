# Proposal: Limit xAI Credential Refresh Storm

- Date: 2026-07-24
- Owner(s): hewei
- Stakeholders: Grok users, production operators
- Status: Accepted

## Context / Problem

Production has roughly 4,800 xAI credential files, including more than 2,500 disabled credentials. Disabled OAuth credentials are still scheduled for automatic refresh, and permanent xAI `invalid_grant` responses are returned as plain HTTP 400 errors. The manager therefore retries those credentials every five minutes. After process restarts, the default 16 refresh workers create a burst against the shared upstream proxy, which itself returns connection failures or `Too Many Requests`; client requests then surface as 429/500/502.

## Goals

- Remove disabled credentials from the automatic refresh schedule.
- Treat xAI `invalid_grant` refresh responses as permanent unauthorized failures for the current process.
- Keep valid enabled OAuth credentials refreshing normally.
- Reduce production refresh concurrency and increase transient-error cooldown without deleting credentials.

## Non-goals

- Do not delete credential files.
- Do not permanently disable rolling 24-hour quota exhaustion.
- Do not change request/response translation or Cloudflare behavior.

## Proposed Approach

Guard both refresh evaluation and scheduler insertion when `Auth.Disabled` is true. Wrap xAI `invalid_grant` refresh errors with an unauthorized status so the existing manager removes them from the refresh schedule. Add focused tests. In production, set two auto-refresh workers and a five-minute transient failure cooldown while keeping the existing bounded request retry settings.

## Risks & Mitigations

- Disabled credentials will no longer have access tokens refreshed in the background.
  - This matches their disabled state; re-enabling them puts them back into the scheduler and refreshes as needed.
- A transient xAI 400 could be misclassified.
  - Match only the explicit case-insensitive `invalid_grant` marker.
- Lower concurrency can delay refresh of a large newly expired set.
  - Two workers preserve progress while protecting the constrained upstream proxy.
