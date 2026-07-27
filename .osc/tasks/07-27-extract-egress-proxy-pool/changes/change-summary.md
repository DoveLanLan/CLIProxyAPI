# Change summary: standalone EgressProxyPool

## Outcome

Created `/root/Projects/Go/src/EgressProxyPool` as a new Git repository and
extracted the Mihomo-backed xAI egress control plane from CLIProxyAPI.

The standalone project now owns:

- rollout and rendezvous lane routing
- per-lane start limits
- provider/node discovery and refresh
- public egress-IP probing and deduplication
- IP/node quarantine and lane rotation
- persisted pool state and counters
- write-only subscription registry and transactional Mihomo reload
- the Mihomo container, controller secret, generated config, and provider cache
- authenticated private status/control/subscription APIs

CLIProxyAPI now owns only the application-aware integration:

- explicit per-auth proxy precedence
- HMAC route-key generation without sending raw auth IDs
- exact xAI spending-limit 402 classification
- replayability and pre-downstream-payload checks
- HTTP/SSE/WebSocket alternate retry orchestration
- request-scoped 503/Retry-After mapping
- compatibility forwarding through the existing Management API routes

## Deployment

`EgressProxyPool/compose.yml` runs a controller and Mihomo on the private named
network `egress-proxy`. No host ports, TUN, host networking, privileged mode, or
added Linux capabilities are used. CLIProxyAPI's xAI overlay now mounts only the
pool API token and joins that external network.

The controller generates the empty fail-closed Mihomo configuration before
Mihomo starts. A five-second startup-only recovery loop handles the deliberate
controller-before-Mihomo boot order and stops once the pool becomes ready.

## Compatibility

- Existing `/v0/management/xai-proxy-pool/**` paths remain available.
- Exact-402 retry, repeated-402 credential failure, preconnect retry, and
  no-midstream-replay semantics remain covered.
- The config shape changes from embedded Mihomo topology to `service-url` and
  `service-token-file`; deployment must start the standalone project first.
- Route keys and token rotation can cause a one-time lane reassignment because
  the standalone service hashes an HMAC digest rather than the raw auth ID.
- No production deployment or live subscription mutation was performed.
