# Rollback Notes: Merge Upstream Main While Preserving Deployment Files

- Date: 2026-04-24
- Related: `proposal.md`, `spec.md`, `tasks.md`

## Rollback point

- Pre-merge local main: `25e4ece2`
- Upstream merged: `f1ba6151`

## Rollback procedure

1. If the merge commit has not been pushed, reset local `main` back to `25e4ece2`.
2. If the merge commit has been pushed, revert the merge commit with `git revert -m 1 <merge-commit>`.
3. Re-run `go build -o test-output ./cmd/server` and any production smoke tests.

## Data/config considerations

- No database migrations were introduced.
- Qwen/IFlow code is removed by the merge; production configs that reference those providers must be updated before deployment.
- Workflow and Docker/deployment files were preserved from the local fork, so rollback of deployment mechanics should not be needed for this merge.

## Risk notes

- Reverting the merge also removes upstream Codex/Responses/auth/model-registry fixes brought in by `upstream/main`.
