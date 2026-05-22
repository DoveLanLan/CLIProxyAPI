# Tasks: Update VPS CPA-Manager Image

- Date: 2026-05-22
- Owner(s): hewei
- Related: proposal.md, spec.md

## Assumptions

- The production GitHub environment has valid SSH secrets.
- The latest deploy has already copied a compose file with `CPA_MANAGER_IMAGE` support to the VPS.

## Checklist

- [x] 1) Add ops workflow
  - Target: `.github/workflows/update-cpa-manager-image.yml`
  - Change: Add automatic one-time and manual-dispatch image update workflow.
  - Verify: `actionlint`.

- [x] 2) Protect remote env update
  - Target: workflow remote script
  - Change: Validate image, back up `.env`, upsert only `CPA_MANAGER_IMAGE`.
  - Verify: script review.

- [x] 3) Restart CPA-Manager only
  - Target: workflow remote script
  - Change: Pull target image and run `docker compose up -d cpa-manager`.
  - Verify: Actions run output.

## Notes

- Target image: `ghcr.io/dovelanlan/cpa-manager:sha-7fa4bfb77b917ddd02141b7fd723182cf2a47013`.
