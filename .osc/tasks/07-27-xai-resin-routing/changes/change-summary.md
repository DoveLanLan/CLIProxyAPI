# Change summary: native xAI Resin routing

## Outcome

CLIProxyAPI can now route all non-overridden xAI credentials through one Resin
forward-proxy configuration. Operators do not create or maintain one Resin
Account or one `proxy_url` per credential.

For every selected xAI auth, CPA calculates HMAC-SHA256 over the stable auth ID
with a CPA-only identity key, truncates the digest to 128 bits, and sends Resin
the V1 proxy username `Platform.xai-<32 lowercase hex characters>`. The raw auth
ID, xAI token, and identity key are not sent to Resin. Resin creates and manages
the Account lease when it first sees that username.

## Runtime behavior

- Routing priority is explicit auth proxy, Resin, EgressProxyPool, then the
  existing CPA global/context transport.
- Resin and EgressProxyPool cannot be enabled together for automatic routing.
- HTTP, SSE, xAI WebSocket, Management API HTTP calls, and OAuth refresh all use
  the derived in-memory proxy URL.
- Refresh restores the original auth proxy before the auth is returned for
  persistence; no generated Resin credentials are written to auth files.
- Invalid enabled settings fail with a request-scoped 503 and never fall back
  to the CPA global proxy.
- A Resin network failure before any response payload is exposed is retried
  once with the same selected xAI auth and the same derived Resin Account. A
  repeated failure and any mid-response failure remain request-scoped 502
  errors, so Resin failures do not cool valid xAI credentials or iterate
  through a large credential pool.
- Explicit auth `proxy_url` and xAI API-key `proxy-url` remain intentional
  overrides and bypass automatic Resin routing.

## Configuration and deployment

The new `xai-resin-proxy` block contains the Resin forward-proxy endpoint,
Platform, proxy-token file, and identity-key file. Secret values remain outside
`config.yaml` and are mounted read-only by
`deploy/compose.production.xai-resin.yml`.

`deploy/scripts/remote-deploy.sh` selects the overlay, validates both files, and
rejects simultaneous Resin/EgressProxyPool deployment flags. The Chinese setup
guide documents the matching Resin V1 environment, shared Docker network,
Platform behavior, secret creation, verification, rotation, and rollback.

## Compatibility and limits

- The feature is disabled by default and does not change non-xAI providers.
- Existing behavior is unchanged while Resin is disabled.
- Resin standard CONNECT cannot inspect xAI's encrypted application-level 402,
  so the EgressProxyPool exact-402 A/B workflow does not run for Resin traffic.
- The first OAuth login occurs before a stable auth ID exists and is outside the
  dynamic identity path; subsequent refreshes use it.
- No files under `internal/translator/**` changed.

## Production rollout

The reviewed build was deployed to `bytevirt` as
`cliproxyapi:xai-resin-20260728`. CPA and Resin are running on the shared
`vps-gateway` network; CPA's HTTP endpoint returns 200 and Resin reports healthy.
The EgressProxyPool backend is disabled and Resin routing is enabled.

The existing Resin V1 proxy token is mounted into CPA together with a new
CPA-only identity key. Both source files are mode `0600` and both container
mounts are read-only. No xAI auth file was modified and no per-credential Resin
Account was provisioned manually.

Real CPA xAI requests created 89 valid `xai-<32 lowercase hex>` leases in the
`Default` Platform, with no malformed xAI leases. The sampled credentials then
reached xAI and received the application-level response
`402 personal-team-blocked:spending-limit`. This is an upstream credential
spending/subscription condition, not a CPA or Resin routing failure.

The pre-rollout production backup is
`/opt/cliproxyapi/backups/xai-resin-20260727T223545Z`.

## Failover hardening rollout

The follow-up build was deployed as
`cliproxyapi:xai-resin-retry-20260728`. Its immediate rollback point is
`/opt/cliproxyapi/backups/xai-resin-retry-20260728T121546Z`, which contains the
previous image reference and production configuration files.

A scoped production fault rejected exactly one outbound connection from a
leased Resin node. Resin recorded `UPSTREAM_CONNECT_FAILED` at `connect_dial`,
opened that node's circuit at failure count one, and moved the same anonymous
Account from node `843d77ea9214...` to `3b27e4da34f4...`. CPA replayed the
request internally with the same auth and Account; the management client and
xAI upstream both returned HTTP 200 instead of exposing the first Resin 502.
The temporary network rule was removed, and a successful egress probe restored
the intentionally failed node.

After fault injection, 20 consecutive real `grok-4.5` requests completed with
HTTP 200, zero HTTP 502 responses, and a 3.665-second mean duration. CPA and
Resin remained HTTP 200/healthy.
