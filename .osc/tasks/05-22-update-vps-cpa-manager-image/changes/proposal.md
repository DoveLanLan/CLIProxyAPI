# Proposal: Update VPS CPA-Manager Image

- Date: 2026-05-22
- Owner(s): hewei
- Stakeholders: CLIProxyAPI operators
- Status: Accepted

## Context / Problem

The local machine cannot SSH directly into the production VPS, but the existing GitHub Actions production environment already has the deployment SSH key. The VPS needs `/opt/cliproxyapi/.env` updated with a pinned CPA-Manager fork image.

## Goals (Why/What)

- Update `CPA_MANAGER_IMAGE` on the VPS without printing `.env` secrets.
- Restart only the `cpa-manager` service after the image change.
- Keep the workflow reusable for future pinned image updates.

## Constraints

- Use the existing production SSH secrets.
- Do not modify management secrets or other `.env` values.
- Do not restart the full CLIProxyAPI stack unnecessarily.

## Non-goals

- No CPA-Manager source changes.
- No CLIProxyAPI runtime code changes.

## Proposed Approach (high-level)

Add a production ops workflow that upserts only `CPA_MANAGER_IMAGE` in `/opt/cliproxyapi/.env`, backs up the original `.env`, pulls the target CPA-Manager image, and runs `docker compose up -d cpa-manager`.

## Risks & Mitigations

- Risk: Incorrect image input could point production at an unexpected registry.
  - Mitigation: Validate image inputs against `ghcr.io/dovelanlan/cpa-manager:*`.
- Risk: `.env` update could disturb secrets.
  - Mitigation: Use an awk upsert for only one line and create a timestamped backup.

## Open Questions

- None.
