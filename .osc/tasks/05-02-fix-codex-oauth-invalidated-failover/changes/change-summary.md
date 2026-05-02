# Change Summary: Fix Codex OAuth Invalidated Token Failover

- Date: 2026-05-02
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`, `tasks.md`

## What changed

- Added Codex identity metadata extraction for `chatgpt_account_id`, `chatgpt_user_id`, `codex_account_hash`, `codex_user_hash`, and `plan_type` while keeping `account_id` for the `Chatgpt-Account-Id` header.
- Updated Codex auth synthesis/load/refresh and management OAuth save paths to carry the new non-secret metadata.
- Added exact `401` + `invalidated oauth token` handling that persistently disables only the failing Codex auth and continues failover in the same request, bypassing `max-retry-credentials` only for that quarantined credential.
- Added regressions for same-team distinct Codex users, invalidated-token quarantine/failover, and generic `401` cooldown behavior.

## Why

Team/workspace Codex accounts can share `chatgpt_account_id` across multiple human users, so that value must not be treated as a unique user identity. Invalidated OAuth tokens are unrecoverable for that auth file and should be isolated without blocking healthy Codex credentials.

## Notable decisions

- The invalidated-token branch matches only Codex provider errors with HTTP `401` and a message containing `invalidated oauth token`.
- Generic `401` continues to use the existing temporary model cooldown path.
- No `internal/translator/**` files, public API routes, or config keys were changed.
