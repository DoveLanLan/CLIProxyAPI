# Proposal: Handle xAI streamed free-usage exhaustion and repair the Grok account pool

- Date: 2026-07-23
- Owner(s): hewei
- Stakeholders: CLIProxyAPI operators and Claude Code users
- Status: Accepted

## Context / Problem

Production xAI free-tier quota errors frequently arrive inside an HTTP 200 SSE
response. The HTTP executor translates those error objects as ordinary stream
payloads, so the auth manager cannot cool the credential or fail over before
Claude Code receives the error. Separately, the production inspection job
permanently disables rolling-quota accounts and never revisits them, causing the
credential pool to shrink monotonically.

## Goals (Why/What)

- Convert streamed xAI free-usage exhaustion into an auth-manager-visible 429.
- Preserve the existing 24-hour cooldown and cross-credential failover design.
- Persist production cooldown state and distribute traffic across healthy free accounts.
- Reinspect disabled accounts and delete only credentials proven permanently unusable.
- Restore healthy recoverable credentials without exposing tokens or account identities.

## Constraints

- Do not add read timeouts after an upstream connection is established.
- Do not modify `internal/translator/**`.
- Do not log, print, or copy management keys or credential contents.
- Treat rolling quota exhaustion and probe/network errors as recoverable.
- Back up deletion targets before any irreversible production action.
- Keep CPU and probe concurrency safe for the two-core production host.

## Non-goals

- Purchasing SuperGrok or changing the upstream quota policy.
- Reworking unrelated provider retry behavior.
- Blanket-enabling or blanket-deleting all currently disabled credentials.

## Proposed Approach (high-level)

Recognize xAI SSE error objects in the HTTP executor before protocol translation
and surface free-usage exhaustion through the existing `statusErr`/auth cooldown
path. Verify credential rotation with focused tests. Then enable persisted
cooldowns, change production routing from fill-first to round-robin, and adjust
inspection policy so rolling quota is recoverable. Reinspect disabled accounts,
create a restricted backup of hard-dead targets, delete only permanent failure
classes, and re-enable credentials that pass a fresh health probe.

### Follow-up: persist the production inspection unit

The production service unit was originally installed from an untracked local
draft. Its `GROK_INSPECT_DISABLE_CLASSES` override omitted
`permission_denied`, leaving freshly detected spending-limit accounts enabled
despite the tracked runner's safe default. Track the service and timer under
`deploy/systemd/`, install them from the production deploy script, and keep the
permanent class list aligned with the runner default.

## Risks & Mitigations

- Risk: A normal response containing an `error` field could be misclassified.
  - Mitigation: Match explicit xAI error shapes and add negative tests.
- Risk: Re-enabling a truly invalid credential could reintroduce failures.
  - Mitigation: Re-enable only freshly inspected `healthy` results.
- Risk: Deleting credentials is irreversible.
  - Mitigation: Back up exact targets with mode 0600 and record counts/checksums.
- Risk: A full disabled-account scan could load the VPS or upstream.
  - Mitigation: Keep three workers, avoid overlapping runs, and monitor resources.

## Open Questions (max 3)

- None blocking; deletion classes are defined conservatively in the spec.
