# Proposal: standalone EgressProxyPool

## Problem

The current xAI feature mixes provider protocol handling with a reusable egress
control plane. Subscription changes, Mihomo upgrades, and node operations require
shipping CLIProxyAPI even though they do not change its public AI APIs.

## Decision

Extract the control plane into a sibling `EgressProxyPool` project. Use a narrow,
authenticated HTTP API for route acquisition and outcome reporting. Keep xAI
response interpretation and replay inside CLIProxyAPI.

## Goals

- Independently maintain subscription nodes and Mihomo versions.
- Remove Mihomo controller credentials and writable Mihomo configuration from
  CLIProxyAPI.
- Preserve fail-closed routing and current exact-402 behavior.
- Preserve CLIProxyAPI Management API compatibility during migration.
- Keep active data traffic directly between CLIProxyAPI and Mihomo; the control
  service is not a bandwidth proxy.

## Non-goals

- TLS interception or response inspection in the proxy project.
- Host networking, TUN, privileged containers, or published proxy ports.
- Supporting providers other than the current xAI policy in this migration.
- A browser UI.

## Risks

- Control API unavailability: enrolled requests fail closed with request-scoped
  503 rather than falling back to a known-blocked direct IP.
- Cross-service contract drift: version all endpoints under `/v1` and add client
  protocol tests.
- Orphaned probe locks: use random single-use lease IDs with expiration.
- Secret exposure: bearer token files, bounded bodies, redacted errors, and no
  auth IDs or subscription URLs in status responses.
