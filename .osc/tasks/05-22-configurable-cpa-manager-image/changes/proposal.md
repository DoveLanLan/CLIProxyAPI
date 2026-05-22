# Proposal: Make CPA-Manager Image Configurable

- Date: 2026-05-22
- Owner(s): hewei
- Stakeholders: CLIProxyAPI operators, CPA-Manager fork maintainers
- Status: Accepted

## Context / Problem

The production stack currently pins the CPA-Manager service to `seakee/cpa-manager:latest`. That makes it awkward to test or deploy a maintained CPA-Manager fork for usage dashboard fixes because the operator must edit compose directly on the VPS.

## Goals (Why/What)

- Let production deployments point CPA-Manager at a forked image from `.env`.
- Preserve the current default image and port/data-volume topology.
- Document the recommended fixed-tag deployment pattern for a fork.

## Constraints

- Keep the default behavior unchanged.
- Do not vendor CPA-Manager source into this repository.
- Do not change CPA-Manager ports, volumes, credentials, or usage query limit behavior.
- Keep management endpoints private to the Tailscale deployment path.

## Non-goals

- No CPA-Manager source changes in this repository.
- No changes to CI image publishing for CLIProxyAPI.
- No data migration or SQLite schema change.

## Proposed Approach (high-level)

Parameterize only the production CPA-Manager service image with `CPA_MANAGER_IMAGE`, defaulting to `seakee/cpa-manager:latest`. Document that forked production deployments should pin a GHCR `sha-<commit>` tag while keeping the existing `18318` mapping and `/data` volume.

## Risks & Mitigations

- Risk: Operators could point at an incompatible image.
  - Mitigation: Keep the default image unchanged and document fixed known-good tags for fork usage.
- Risk: `latest` on a fork can make rollbacks unclear.
  - Mitigation: Recommend `sha-<commit>` or explicit version tags for VPS production.

## Open Questions

- None.
