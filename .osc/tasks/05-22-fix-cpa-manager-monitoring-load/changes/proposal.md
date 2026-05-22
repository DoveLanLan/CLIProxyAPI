# Proposal: Fix CPA-Manager Monitoring Load

- Date: 2026-05-22
- Owner(s): hewei
- Stakeholders: VPS operators, CPA-Manager users
- Status: Proposed

## Context / Problem

The CPA-Manager monitoring page on the VPS stays behind a loading overlay even though most network requests succeed. Browser inspection shows `/status` and metadata endpoints return, while `/v0/management/usage` and `/v0/management/usage/export` hang or time out. The panel auto-refresh interval is shorter than the usage request timeout, so repeated slow usage requests keep the loading state active.

## Goals (Why/What)

- Make the deployed CPA-Manager monitoring page load usable data instead of querying the full usage database on every refresh.
- Keep the mitigation deployment-only and avoid changing upstream CPA-Manager source in this repository.
- Document the upstream root causes for follow-up.

## Constraints

- Do not touch `.github` or Docker build files for this task.
- Keep management access private and preserve the existing CPA-Manager service topology.
- Avoid exposing management keys or captured request credentials in logs or documentation.
- Keep the change reversible through environment configuration.

## Non-goals

- Vendoring or forking CPA-Manager source into CLIProxyAPI.
- Changing CLIProxyAPI protocol translators.
- Replacing SQLite data or deleting existing usage history automatically.

## Proposed Approach (high-level)

Set an explicit CPA-Manager Usage Service query limit in the production compose environment so `/v0/management/usage` reads a small bounded recent window instead of the default 50,000 events. This is a deployment-side mitigation for the currently deployed external service. The deeper upstream fixes remain in CPA-Manager: make usage requests cancel/settle cleanly under auto-refresh and avoid reading large raw event payloads for the dashboard response.

## Risks & Mitigations

- Risk: Lower query limits reduce all-time dashboard coverage.
  - Mitigation: Make the limit configurable via `.env` with a conservative default suitable for live monitoring.
- Risk: Existing large SQLite rows can still make even a bounded query slow.
  - Mitigation: Document a VPS rollback/remediation path to lower the limit further or prune/rotate the CPA-Manager data volume manually.
- Risk: This does not fix CPA-Manager upstream behavior.
  - Mitigation: Record the exact upstream symptoms and likely fixes for an upstream issue/PR.

## Open Questions (max 3)

- None.
