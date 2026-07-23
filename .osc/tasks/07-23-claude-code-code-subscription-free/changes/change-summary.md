# Change Summary: xAI streamed free-usage failover and Grok account-pool repair

- Date: 2026-07-23
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`, `tasks.md`

## What changed

- The xAI HTTP streaming executor now recognizes explicit SSE error objects before protocol translation.
- `subscription:free-usage-exhausted` from an HTTP 200 stream is surfaced as a 429-style executor error with the existing 24-hour retry hint, allowing the auth manager to cool the credential and fail over.
- Regression coverage proves normal SSE events still translate and a streamed free-usage error rotates to a second credential.
- The tracked Grok inspection runner excludes rolling quota and transient probe failures from permanent disable, and can optionally recover freshly healthy disabled credentials.
- Production now uses persisted cooldown state and round-robin routing. The deployed container runs revision `0672a88e4412aa2d3cc2c8697cdc963f0acc7a72`.

## Production remediation

- Fresh disabled-only inspection classified 2995 credentials: 1346 healthy, 55 quota exhausted, 92 probe errors, 1023 permission denied, and 479 reauth.
- The 479 initial reauth credentials were backed up, checksum-verified, deleted, and verified absent.
- All 1346 healthy credentials were re-enabled after the image and config rollout. A follow-up inspection found two newly reauth credentials; they were separately backed up and deleted.
- Final recovery-set state was 1344 present, 1344 enabled, and 0 disabled. Final observed xAI aggregate was 3232 total, 1876 active, and 1356 disabled.
- Backup archives contain 481 deleted files in total. Backup directories and artifacts use mode 0700/0600, checksums pass, and no target source files remain.

## Notable decisions

- Quota exhaustion and probe/network errors remain recoverable and are never deletion criteria.
- Permanent deletion requires a fresh hard-dead classification and a verified backup.
- The production host keeps three inspection workers and the five-minute timer to protect the two-core VPS.
- No `internal/translator/**` files or public API contracts changed.

## Remaining observation

No `/v1/messages` traffic occurred in the final ten-minute production window. The deployed configuration is ready to persist `.cds` files, but file creation and user-visible retry improvement must be confirmed when real traffic next encounters a cooled credential.
