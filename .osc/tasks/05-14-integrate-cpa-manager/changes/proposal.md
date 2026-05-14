# Proposal: Integrate CPA-Manager Panel and Usage Monitoring

- Date: 2026-05-14
- Owner(s): hewei
- Stakeholders: CLIProxyAPI operators
- Status: Accepted

## Context / Problem

The current repository serves an external `management.html` panel from the default `router-for-me/Cli-Proxy-API-Management-Center` release. CPA-Manager provides a replacement single-file panel and an optional Usage Service for persistent request monitoring. Replacing only the panel is not enough for request monitoring because CPA-Manager expects a usage queue endpoint that this local branch does not have yet.

## Goals (Why/What)

- Serve CPA-Manager as the default `/management.html` panel.
- Add the minimal backend usage queue and management endpoints needed by CPA-Manager request monitoring.
- Provide compose/deployment guidance for a separate CPA-Manager Usage Service.

## Constraints

- Keep changes small; do not upgrade the whole repository to a newer upstream version.
- Do not touch `internal/translator/**`.
- Keep management APIs behind the existing management secret and local/remote access rules.
- Keep Usage Service as an external service with persistent SQLite storage, not a bundled frontend app in this repo.

## Non-goals

- No full upstream v6.10/v7 merge.
- No vendoring CPA-Manager React or Usage Service source into this repository.
- No public exposure of management endpoints in production deploy docs.

## Proposed Approach (high-level)

Backport the minimal usage queue behavior from upstream v6.10.8 into local packages, wire two management endpoints, switch the management panel GitHub release source to CPA-Manager, and update compose/deploy examples so operators can run `seakee/cpa-manager` as the single Usage Service consumer.

## Risks & Mitigations

- Risk: queue payloads may miss fields needed by CPA-Manager analytics.
  - Mitigation: emit stable provider/model/auth/token/latency/failure fields and add focused tests.
- Risk: management panel update fallback could unexpectedly restore the old UI.
  - Mitigation: align the default repository and fallback page with CPA-Manager.
- Risk: multiple Usage Service consumers can drain the same queue.
  - Mitigation: document one consumer per CPA instance and keep queue retention configurable.

## Open Questions

- None.
