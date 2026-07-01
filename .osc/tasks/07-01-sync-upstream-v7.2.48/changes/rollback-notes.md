# Rollback Notes: Sync Upstream v7.2.48 Preserving Deploy and CPA-Manager

- Date: 2026-07-01
- Related: `proposal.md`, `spec.md`, `tasks.md`

## Rollback point

- Pre-sync local `main`: `91dea825` ("Improve Docker compose image rebuild")
- Upstream target: `956ce7cf` (tag `v7.2.48`)
- Last sync point (already in tree as patch, not merge commit): `21fad9db`

## Rollback procedure

1. If the sync has not been committed: discard the working-tree changes on the sync branch (`git checkout -- .` + remove untracked upstream-added files, or `git reset --hard 91dea825` on the temp branch) and return `main` to `91dea825`.
2. If the sync has been committed to `main` but not pushed: `git reset --hard 91dea825`.
3. If the sync has been pushed: `git revert -m 1 <sync-commit>` (or revert the sync commit series) on `main`.
4. Re-run `go build -o test-output ./cmd/server` and the focused behavior tests to confirm the fork is back to pre-sync state.

## Data/config considerations

- No database or storage migrations introduced by this sync.
- Protected deployment/Docker/CI files are unchanged, so deployment mechanics need no rollback.
- CPA-Manager defaults are preserved through the sync, so no panel-config rollback is needed.
- If upstream-added config fields were already written into a production `config.yaml`, those fields become no-ops after rollback but should be cleaned up.

## Safer Alternative To Full Rollback

- If a specific upstream feature regresses, revert or patch the affected package while keeping the rest of the v7.2.48 sync.
- If a new provider/plugin path fails operationally, disable or exclude that provider/plugin in config first, then patch the targeted executor/handler with a focused regression test.
- If a local behavior patch (Codex failover, xhigh, null usage, DeepSeek, GPT-5.5) regressed during re-application, fix that single patch and re-run its focused test rather than rolling back the whole sync.

## Risk notes

- Rolling back removes upstream plugin system, signature, safemode, homeplugins, image/video, `gpt-image-1.5`, `disable-cooling`, `max` reasoning, `ResetQuota`, Codex WS↔SSE replay, and translator/runtime/registry/auth fixes brought in by `v7.2.48`.
