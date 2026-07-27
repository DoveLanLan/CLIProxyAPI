# Refactor: Extract standalone egress proxy pool

## Current state

CLIProxyAPI currently owns xAI request semantics and the complete Mihomo control
plane: subscriptions, generated configuration, provider refresh, egress probing,
lane selection, quarantine, persistence, and operator APIs. This couples proxy
node maintenance to CLIProxyAPI releases and substantially increases the fork's
merge surface.

## Target architecture

Create `/root/Projects/Go/src/EgressProxyPool` as an independently buildable and
deployable Go project. It owns the Mihomo sidecar and all proxy-pool state. The
CLIProxyAPI xAI executor calls its authenticated private API for routes, probe
leases, failure observations, status, and subscription operations.

CLIProxyAPI continues to own exact xAI 402 classification and safe HTTP/SSE/
WebSocket replay decisions because a CONNECT proxy cannot inspect encrypted xAI
responses or know whether downstream payload has already been emitted.

## Impact

- New project: `/root/Projects/Go/src/EgressProxyPool`
- CLIProxyAPI: config, xAI executor pool client, Management API compatibility,
  deployment overlay, tests, and documentation
- API: versioned private `/v1` control API between the two services
- Storage: pool state, subscriptions, generated Mihomo config, and controller
  secret move out of CLIProxyAPI-owned volumes

## Regression scope

- Explicit auth proxy still bypasses the pool.
- Missing remote pool fails closed for enrolled xAI requests.
- Exact pre-response 402 still performs one same-auth alternate attempt.
- Repeated exact 402 remains a credential failure.
- Mid-response failures are observed but never replayed.
- Existing Management API routes remain available as a compatibility facade.
- Subscription URLs remain write-only and redacted.

## Phased plan

1. Create the standalone project and move the tested pool/controller/subscription
   implementation into it.
2. Add an authenticated control API with bounded request bodies and expiring
   probe leases.
3. Replace the embedded CLIProxyAPI pool with a remote client while preserving
   the executor-facing contract.
4. Move Docker ownership and update examples/documentation.
5. Run both projects' tests, required CLIProxyAPI build, and security checks.
