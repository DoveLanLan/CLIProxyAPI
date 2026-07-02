# Journal 14: Remove Legacy CPA-Manager Production Service

- Date: 2026-07-02
- Task: `.osc/tasks/07-01-sync-upstream-v7.2.48`

## Summary

GitHub Actions production deploy failed at `Deploy production stack` because the legacy `cpa-manager` compose service attempted to bind `100.67.99.9:18318`, but production now uses CPA Manager Plus and that port is already allocated outside this stack.

## Decisions

- Remove the legacy standalone `cpa-manager` service from `deploy/compose.production.yml`.
- Keep production management access on the existing CLIProxyAPI private Tailscale port `18317`.
- Delete `.github/workflows/update-cpa-manager-image.yml` because it only updates/restarts the retired service.
- Leave local development compose files untouched; this is scoped to production deploy assets.
- Keep remote `data/cpa-manager/` data untouched. If cleanup is needed, do it manually on the VPS after confirming no external service needs it.

## Validation

- `docker compose -f deploy/compose.production.yml --env-file deploy/.env.example config --services` returned `cli-proxy-api`.
- `docker compose -f deploy/compose.production.yml -f deploy/compose.production.split-proxy.yml --env-file deploy/.env.example config --services` returned `split-proxy` and `cli-proxy-api`.
- Stale production reference scan for the retired service returned no matches.
- `.github/workflows/update-cpa-manager-image.yml` is absent.
- `go build -o test-output ./cmd/server && rm test-output` passed.
- `git diff --check` passed.

## Risks

- If an external manually-created container still owns `18318`, it is no longer a deploy blocker because this stack no longer binds that port.
- If the old `cpa-manager` container was created by this compose project, the next deploy should remove it through `docker compose up -d --remove-orphans`.
- If the old container was not created by this compose project, manual VPS cleanup may still be needed.

## Rollback

Revert the hotfix commit or restore the changed deploy/workflow files from the previous revision. Before rerunning production deploy after rollback, verify that host port `18318` is free, otherwise the original failure will return.
