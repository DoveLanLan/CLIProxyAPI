# Proposal: xAI dedicated rotating proxy pool

## Context / Problem

Grok Free requests from the production VPS can receive HTTP 402
`personal-team-blocked:spending-limit` after sustained traffic. Controlled
testing showed the same credential succeeds when only the public egress IP is
changed, so treating every such response as a permanent credential failure can
disable healthy accounts. The current global/per-auth proxy model has no
provider-scoped pool, stable lane assignment, or egress-aware A/B retry.

## Goals

- Add an opt-in, xAI-only proxy pool that works for the server and embeddable SDK.
- Keep credentials stable on a bounded set of egress lanes.
- Separate credential failures, xAI IP blocks, and ordinary proxy-node failures.
- Rotate only new connections on an affected lane and preserve active streams.
- Support multiple Mihomo proxy providers without storing subscription secrets
  in the repository or CPA config.
- Provide redacted status and limited operator controls through Management API.
- Let an authenticated operator create, replace, disable, inspect, and safely
  delete multiple subscription sources without routine VPS SSH access.
- Apply enabled subscription mutations transactionally through Mihomo hot reload
  and preserve the previous registry/runtime configuration on failure.

## Constraints

- Explicit auth proxies remain the highest-priority manual override.
- The xAI pool must not chain through the global `proxy-url`.
- No timeouts may be added after an upstream connection is established.
- `internal/translator/**` must remain untouched.
- Pool-local errors must not permanently cool or disable credentials.
- Production installation, service restart, and live subscription data are out
  of scope until separately authorized.

## Non-goals

- Host-wide TUN routing or proxying providers other than xAI.
- Automatic lane-count scaling, a web dashboard, or external notifications.
- TLS interception or disabling xAI certificate validation.
- Guaranteeing that xAI will not apply account-, ASN-, or fingerprint-level
  enforcement in the future.
- A subscription-management web UI or standalone CLI helper in the first
  release.
- Arbitrary Mihomo YAML upload, custom subscription headers, or caller-selected
  filesystem paths.

## Proposed Approach

Add an `xai-proxy-pool` config block containing lane listener URLs, Mihomo
selector names, a probe listener/selector, controller secret-file reference,
state path, rollout percentage, rate limits, and quarantine durations. The
registered `XAIAutoExecutor` owns one pool runtime and applies it to HTTP, SSE,
WebSocket, refresh, and generic xAI `HttpRequest` paths.

The runtime uses rendezvous hashing for stable auth-to-lane assignment. It
queries Mihomo's private API for providers, node health, and selector state;
discovers actual public egress IPs through the probe listener; deduplicates and
quarantines by egress IP; and persists only non-sensitive state. Exact
pre-response 402 failures receive one same-auth retry through the probe route.
Success confirms an IP block and promotes the probe node into the affected
lane. A repeated exact 402 remains a credential failure. Unknown verification
or pool exhaustion returns a request-scoped 503.

Management endpoints expose redacted status, provider refresh, lane rotation,
network checks, and manual quarantine controls. A secret-free Compose overlay
and Mihomo example show the production topology without enabling it.

For subscription management, CPA owns a mode-`0600` write-only registry as the
sole provider source of truth and renders the fixed Mihomo topology with enabled
providers. Create/update/enable operations use candidate configuration reload,
provider discovery, and rollback as one serialized transaction. URLs never
leave write requests; list/status responses contain safe metadata and a short
fingerprint only. Deletion requires the provider to be disabled and absent from
all active lanes.

## Risks & Mitigations

- **Retry duplicates work:** only retry before any downstream payload and only
  for the exact 402 or a confirmed pre-connect node failure.
- **Rotation interrupts streams:** update selectors for future connections and
  close only idle HTTP transports; never delete Mihomo connections.
- **Duplicate subscription exits:** resolve candidate egress IPs through the
  probe listener and reject duplicates.
- **Controller compromise:** keep the API on the private Docker network, require
  a secret file, and expose only narrow operations from CPA.
- **CPU/memory pressure:** use one Mihomo process, bounded candidate scans,
  per-lane rate limiting, and documented 1 CPU / 384 MiB deployment limits.
- **Backward compatibility:** default disabled, no auth-file migration, and
  legacy proxy precedence restored on explicit disable.
- **Bad subscription breaks the pool:** require successful download, parsing,
  provider discovery, and lane reconciliation before committing an enabled
  mutation; reload the previous payload on any failure.
- **Secret exposure through management:** make URLs write-only, cap input size,
  suppress raw fetch/controller errors, and never log mutation bodies.
- **Concurrent lost updates:** serialize mutations and commit the registry only
  after the candidate running configuration is verified.
