# Rollback Notes: Fix Codex OAuth Invalidated Token Failover

- Date: 2026-05-02
- Related: `spec.md`, `tasks.md`

## Rollback strategy

Revert the code and test changes in this task, then rerun the focused auth packages and `go build -o test-output ./cmd/server && rm test-output`.

## Data / migration considerations

- No schema migration is involved.
- Auth JSON files may contain optional new non-secret metadata fields. Old code should ignore unknown JSON fields, but persisted `disabled: true` entries from invalidated-token quarantine would remain disabled until re-enabled or re-login rewrites the auth file.

## Operational notes

- Watch Codex auth-file status for `codex oauth token invalidated; re-login required`.
- Rollback does not restore invalidated tokens; affected users still need to re-login.
