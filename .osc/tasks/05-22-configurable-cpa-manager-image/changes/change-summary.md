# Change Summary: Make CPA-Manager Image Configurable

- Date: 2026-05-22
- Owner(s): hewei
- Related: proposal.md, spec.md, tasks.md

## What changed

- `deploy/compose.production.yml` now reads the CPA-Manager image from `CPA_MANAGER_IMAGE`, defaulting to `seakee/cpa-manager:latest`.
- `deploy/README.md` documents `CPA_MANAGER_IMAGE` and recommends fixed fork tags for VPS production deployments.

## Why

This lets operators deploy a maintained CPA-Manager fork by changing `/opt/cliproxyapi/.env` instead of editing compose, while preserving the existing upstream default, private `18318` mapping, `/data` volume, and usage query limit protection.

## Notable decisions

- Keep CLIProxyAPI responsible only for referencing the image; CPA-Manager fork source and image publishing stay in the fork repository.
- Recommend `sha-<commit>` or explicit version tags for production and reserve `latest` for manual testing.
