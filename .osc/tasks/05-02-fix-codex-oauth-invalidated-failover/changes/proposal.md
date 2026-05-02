# Proposal: Fix Codex OAuth Invalidated Token Failover

- Date: 2026-05-02
- Owner(s): hewei
- Stakeholders: Codex OAuth users, local operators, SDK embedders
- Status: Accepted

## Context / Problem

Codex OAuth auth metadata currently treats the `chatgpt_account_id`-derived value as the `account_id`. For team/workspace accounts, that ID identifies the ChatGPT account/workspace rather than the human user, so multiple valid users in the same team can share the same derived filename prefix. Separately, an upstream `401` response containing `invalidated oauth token` should permanently quarantine only that credential and immediately fail over to another Codex auth, while generic `401` responses should keep the existing temporary cooldown behavior.

## Goals (Why/What)

- Preserve `account_id` for `Chatgpt-Account-Id` header compatibility while adding explicit Codex account/user metadata.
- Keep same-team Codex OAuth files with different human users active and distinct.
- Persistently disable only credentials that receive the exact invalidated-token error class.
- Continue the current request with another available Codex auth, including when `max-retry-credentials` is `1`.

## Constraints

- Do not modify `internal/translator/**`.
- Do not log access tokens, refresh tokens, or full JWT values.
- Existing Codex auth files without new metadata must remain valid.
- Generic `401` handling must remain a temporary cooldown path.

## Non-goals

- No config, API route, or translator changes.
- No delete-by-team-prefix workflow changes.
- No fix for unrelated proxy scheme errors.

## Proposed Approach (high-level)

Enrich Codex auth file metadata during OAuth file synthesis by parsing safe identity fields from `id_token`, then adjust auth manager error classification so `401 invalidated oauth token` disables and persists only the failing auth before continuing credential rotation. Add focused tests for metadata distinctness, invalidated-token quarantine/failover, max retry bypass for quarantined credentials, and generic `401` cooldown behavior.

## Risks & Mitigations

- Risk: Disabling the wrong credential would interrupt healthy users.
  - Mitigation: Match only status `401` plus the exact invalidated-token message substring, and add regression tests.
- Risk: New metadata could break existing auth files.
  - Mitigation: Add optional fields only and keep existing `account_id` behavior.
- Risk: Retry changes could bypass configured limits too broadly.
  - Mitigation: Bypass `max-retry-credentials` only for unrecoverable credential invalidation.

## Open Questions (max 3)

- None.
