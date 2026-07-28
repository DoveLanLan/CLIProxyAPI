# Spec: bounded xAI retry through Resin lease rotation

## Configuration contract

```yaml
xai-resin-proxy:
  enabled: true
  proxy-url: "http://resin:2260"
  platform: "Default"
  proxy-token-file: "/run/secrets/resin-proxy-token"
  identity-key-file: "/run/secrets/resin-identity-key"
  admin-url: "http://resin:2260"
  admin-token-file: "/run/secrets/resin-admin-token"
  max-402-retries: 2
```

- `max-402-retries` is the number of additional attempts after the initial
  exact 402. It defaults to zero and is clamped to a maximum of five.
- A positive retry count requires a valid `admin-url` and a readable, non-empty
  `admin-token-file`. Zero retries do not require either field.
- `admin-url` accepts only `http` or `https`, must not contain userinfo, query,
  or fragment data, and may contain a base path.
- Existing proxy token, identity key, Platform, and proxy URL validation remains
  unchanged.

## Stable identity and lease contract

The existing HMAC derivation remains authoritative. Both proxy routing and lease
rotation derive the same Account from the selected auth ID. Healthy traffic
keeps that Account sticky.

On rotation, CPA:

1. Resolves the configured Platform name with `GET /api/v1/platforms`.
2. Caches the matched Platform ID in memory.
3. Calls `DELETE /api/v1/platforms/{platform-id}/leases/{account}` with
   `Authorization: Bearer <admin-token>`.
4. Treats 204 as success and 404 as an already-absent lease.
5. Clears and resolves the cached Platform ID once if Resin indicates that the
   cached Platform no longer exists.

All dynamic path components are URL-escaped. Admin responses are bounded and
must not be copied verbatim into client errors or logs.

## Retry contract

Only an error satisfying both conditions is eligible:

- HTTP status is 402; and
- the structured or compact JSON error code is exactly
  `personal-team-blocked:spending-limit`.

For each eligible attempt before the configured limit:

1. Confirm no downstream payload has been emitted.
2. Delete the current Account lease.
3. Start a new upstream connection with the same xAI auth and stable Resin
   Account.

If lease deletion fails, return a request-scoped 503 without selecting another
xAI credential, invoking EgressProxyPool, or falling back to the global proxy.
If the final attempt still returns the exact 402, return that original class of
upstream error unchanged. Resin may select the same IP again; CPA does not claim
that each retry guarantees a distinct public IP.

## Runtime coverage

- Non-stream `Execute`: retry exact returned 402 errors.
- Generic `HttpRequest`: close each rejected response before retry; retry only
  when the request body is nil or can be recreated through `GetBody`.
- SSE: inspect existing bootstrap chunks; rotate only when exact 402 appears
  before the first non-empty payload. Never replay after payload.
- WebSocket: rotate only when the upstream handshake returns the exact 402,
  before a WebSocket connection is established.
- OAuth refresh: apply the same bounded pre-response retry and restore the
  original persisted proxy after the transient route is removed.
- Mid-response network errors and exact 402-like chunk errors are surfaced and
  never replayed.

Each attempt must construct a fresh request transport or WebSocket dial. No read
timeout may be added after an upstream connection is established.

## Observability and error handling

- Rotation failures use the existing request-scoped Resin 503 error class and
  managed Retry-After behavior.
- Optional debug/warn logs may state that a lease rotation attempt occurred or
  failed, but must omit Account, auth ID, tokens, credentialed URLs, and raw
  admin response bodies.
- Config diffs may show URL and retry-count changes but must redact admin token
  file paths in the same style as existing Resin secret files.

## Deployment assets

- Mount a third read-only Resin admin-token secret into CPA.
- Add the configuration fields to the production Resin Compose overlay, deploy
  script, `.env` template, `config.example.yaml`, and Chinese operator guide.
- The documented recommended value is two additional retries. Existing
  deployments remain behaviorally unchanged until configured.
