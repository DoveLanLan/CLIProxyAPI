# Tech notes

## Architecture decisions

- Sync method: patch-apply with three-way fallback (precedent: `05-22-merge-upstream-non-docker-changes`, `04-24-merge-upstream-main-preserve-deploy`). Avoid plain `git merge upstream/main` because upstream lacks local `.osc` task/workflow files and would delete local workflow state.
- Protected path set (never take upstream): `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.sh`, `docker-build.ps1`, `docker-compose*.yml`, `.env.cluster.example`, `deploy/**`, `.goreleaser.yml`.
- CPA-Manager anchors: `internal/config/config.go` (`DefaultPanelGitHubRepository = "https://github.com/seakee/CPA-Manager-Plus"`), `internal/managementasset/updater.go` (`defaultManagementReleaseURL` / `defaultManagementFallbackURL` → `seakee/CPA-Manager-Plus`).
- Module path already `v7` on both branches (prior sync handled v6→v7), so no module-path migration this round.
- `internal/translator/**` in scope as part of broader sync, not a translator-only task.

## Risks / mitigations

- Upstream refactored files that carry local behavior patches (`internal/runtime/executor/*`, `sdk/cliproxy/auth/*`, `internal/api/handlers/management/*`, `internal/translator/**`). → Re-apply by intent against new structure; run focused tests per patch.
- Upstream plugin system touches management/config/registry and may collide with CPA-Manager management asset logic. → Take upstream structural changes, then re-assert CPA-Manager defaults; run management-handler tests.
- Large diff (~664 files). → Do sync on temp branch `task/sync-upstream-v7.2.48`; record rollback point `91dea825`; write per-area summary.
- Upstream removals/renames of symbols the fork still references. → Build/tests catch dangling refs; accept upstream removals.

## Rollback plan

- Pre-sync `main`: `91dea825`. See `rollback-notes.md`.
- Uncommitted: discard working tree on sync branch.
- Committed not pushed: `git reset --hard 91dea825`.
- Pushed: `git revert` the sync commit(series).
- Targeted alternative: revert/patch a single regressing package and keep the rest of the sync.
