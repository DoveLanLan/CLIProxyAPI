# Spec: Limit xAI Credential Refresh Storm

- Date: 2026-07-24
- Related: `proposal.md`, `tasks.md`

## Scope

### In scope

- Core auth auto-refresh scheduling for disabled credentials.
- xAI OAuth refresh error classification.
- Focused scheduler/executor tests.
- Production refresh worker and transient cooldown configuration.

### Out of scope

- Credential deletion or automatic recovery of disabled credentials.
- Rolling xAI free-quota policy.
- `internal/translator/**`.

## Acceptance Criteria

1. `Disabled` auths return false from both refresh evaluation and scheduler calculation.
2. Re-enabled valid auths remain eligible for normal refresh.
3. An xAI token endpoint response containing `invalid_grant` exposes HTTP status 401 to the auth manager, which stops rescheduling it.
4. Other xAI refresh failures keep their original transient behavior.
5. Production uses at most two automatic refresh workers and a 300-second transient cooldown.
6. Existing request retry bounds remain `request-retry: 2` and `max-retry-credentials: 4`.
7. Focused tests, affected package tests, and the required server build pass.

## Compatibility

- No public API or config schema additions.
- No credential migration.
- Disabled credentials remain stored and visible to management tooling.
- Re-enabling a credential restores normal scheduler eligibility.
