# Spec: Update VPS CPA-Manager Image

- Date: 2026-05-22
- Owner(s): hewei
- Related: proposal.md, tasks.md

## Repo Snapshot

- Production deployment uses `.github/workflows/deploy-production.yml` with SSH secrets and `/opt/cliproxyapi` as `DEPLOY_ROOT`.
- Runtime compose file is `/opt/cliproxyapi/compose.production.yml`.
- CPA-Manager service name is `cpa-manager`.
- `deploy/compose.production.yml` now supports `CPA_MANAGER_IMAGE`.

## Scope

### In scope

- Add a GitHub Actions ops workflow to update the VPS `.env` CPA-Manager image.
- Restart only the CPA-Manager service.

### Out of scope

- Updating CLIProxyAPI image tags.
- Editing management keys or other runtime secrets.
- Changing CPA-Manager source code.

## Acceptance Criteria (testable)

1. Workflow can run automatically once on push and manually via `workflow_dispatch`. (Verify: inspect triggers)
2. Workflow validates image names under `ghcr.io/dovelanlan/cpa-manager:*`. (Verify: inspect validation)
3. Workflow uses production SSH secrets and does not print `.env`. (Verify: inspect steps)
4. Workflow backs up `.env`, upserts `CPA_MANAGER_IMAGE`, pulls the image, and restarts `cpa-manager`. (Verify: inspect remote script and Actions run)

## Behavior / Requirements

The workflow writes `CPA_MANAGER_IMAGE=ghcr.io/dovelanlan/cpa-manager:sha-317e9498bbefa0aa12b5e96837bfd6a7cbc8e3bc` for push-triggered execution. Manual dispatch can provide a future pinned image tag.

## Edge Cases

- If the VPS compose file does not contain `CPA_MANAGER_IMAGE`, the workflow fails before editing `.env`.
- If GHCR access is private and the VPS is not logged in, the image pull fails without changing other `.env` keys.
- A timestamped `.env` backup remains on the VPS for rollback.

## Compatibility Notes

- Backwards compatibility: removing the `CPA_MANAGER_IMAGE` line or setting it to `seakee/cpa-manager:latest` restores upstream image usage.
- Data/migrations: no data or schema changes.
- Config/flags: updates one optional deployment env var.
