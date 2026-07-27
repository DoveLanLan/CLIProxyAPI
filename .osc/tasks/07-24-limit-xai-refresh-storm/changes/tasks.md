# Tasks: Limit xAI Credential Refresh Storm

- Date: 2026-07-24
- Related: `proposal.md`, `spec.md`

## Checklist

- [x] Diagnose production refresh, inspection, proxy, and request error paths.
- [x] Exclude disabled credentials from refresh evaluation and scheduling.
- [x] Classify explicit xAI `invalid_grant` refresh failures as unauthorized.
- [x] Add focused unit tests for disabled scheduling and xAI refresh classification.
- [x] Set conservative production refresh concurrency and transient cooldown.
- [x] Build and deploy a production image with the scheduler fix.
- [x] Verify refresh volume, inspection state, service health, and Grok request behavior.
- [x] Record quality gates, change summary, and rollback instructions.
