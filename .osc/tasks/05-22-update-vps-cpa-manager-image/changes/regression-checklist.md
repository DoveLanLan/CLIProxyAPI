# Regression Checklist: Update VPS CPA-Manager Image

- Date: 2026-05-22
- Related: spec.md, tasks.md

## Gates

- Workflow lint: `/tmp/codex-bin/actionlint .github/workflows/update-cpa-manager-image.yml`
- Diff hygiene: `git diff --check`

## Manual Checks

- Confirm the Actions run completes successfully.
- Confirm the run reports `cpa-manager image=ghcr.io/dovelanlan/cpa-manager:sha-7fa4bfb77b917ddd02141b7fd723182cf2a47013`.
- Confirm `http://100.67.99.9:18318/management.html#/monitoring` renders after restart.
