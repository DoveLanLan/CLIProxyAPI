# Spec: standalone EgressProxyPool extraction

## Service boundary

EgressProxyPool owns pool configuration, rollout, rendezvous lane assignment,
rate limits, provider/node discovery, Mihomo selector changes, egress probing,
IP/node quarantine, counters, state persistence, subscription registry, generated
Mihomo configuration, and operational status.

CLIProxyAPI owns explicit auth-proxy precedence, xAI response parsing, replayable
body checks, pre-payload HTTP/SSE/WebSocket retry, request-scoped error mapping,
and downstream stream safety.

## Private API

- `POST /v1/routes`
- `POST /v1/probes`, outcome operations under `/v1/probes/{id}`
- failure/counter events under `/v1/events`
- status, provider, lane, and quarantine operations under `/v1`
- subscription CRUD/check under `/v1/subscriptions`
- unauthenticated `GET /healthz`; authenticated pool operations

The route key is an HMAC-SHA256 digest produced by CLIProxyAPI. Responses may
contain the private Mihomo proxy endpoint but never subscription URLs or secrets.

## Compatibility

- Existing CLIProxyAPI `/v0/management/xai-proxy-pool/**` routes remain and
  delegate to EgressProxyPool.
- The main config retains `xai-proxy-pool.enabled` and replaces embedded
  controller/storage/topology settings with `service-url` and
  `service-token-file`.
- The standalone service contains the previous pool defaults and six-lane plus
  one-probe topology.

## Security and lifecycle

- Use constant-time bearer-token comparison.
- Cap JSON bodies and reject unknown fields.
- Probe leases expire and release their selector lock.
- Controller and data-plane ports are Docker-internal by default.
- Do not add read deadlines to established xAI streams.
- Subscription mutation remains transactional with rollback.

## Acceptance criteria

1. The new project independently passes `go test ./...` and builds its server.
2. CLIProxyAPI contains no Mihomo controller or subscription registry runtime.
3. Explicit auth proxy bypass, exact-402 retry, repeated-402 behavior, preconnect
   retry, and no midstream replay remain covered.
4. Remote errors remain request-scoped and carry managed Retry-After guidance.
5. Management subscription responses remain redacted and revision-safe.
6. Compose publishes no control/proxy host ports and grants no elevated Linux
   capabilities.
7. `gofmt -w .`, `go test ./...`, and the required CLIProxyAPI build pass.
