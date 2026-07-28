# Spec: xAI Resin proxy routing

## Configuration contract

```yaml
xai-resin-proxy:
  enabled: false
  proxy-url: "http://resin:2260"
  platform: "Default"
  proxy-token-file: "/run/secrets/resin-proxy-token"
  identity-key-file: "/run/secrets/resin-identity-key"
```

- `proxy-url` accepts HTTP, HTTPS, SOCKS5, or SOCKS5H forward-proxy URLs and must
  not contain userinfo, query, or fragment data.
- `platform` defaults to `Default` and must be compatible with Resin V1 naming.
- `proxy-token-file` contains the exact Resin `RESIN_PROXY_TOKEN`.
- `identity-key-file` is CPA-only and contains at least 32 bytes of stable random
  key material. It is never sent to Resin.

## Identity contract

For auth ID `A` and identity key `K`:

1. Calculate `HMAC-SHA256(K, A)`.
2. Encode the first 16 bytes as lowercase hexadecimal.
3. Set Account to `xai-<32 hex characters>`.
4. Set the proxy username to `<Platform>.<Account>` and password to the Resin
   proxy token using `url.UserPassword`.

The same stable auth ID and key produce the same Account across requests and
restarts. Different auth IDs have negligible collision probability. Rotating
only the Resin proxy token does not change Accounts; rotating the identity key
does.

## Routing precedence

1. A non-empty auth `ProxyURL` is used unchanged.
2. Enabled Resin routing derives an in-memory proxy URL.
3. Otherwise the existing EgressProxyPool route may be selected.
4. If neither applies, existing global/context proxy behavior remains.

Enabled Resin plus enabled EgressProxyPool is an invalid automatic-routing
combination and fails closed for auths without an explicit proxy override.

## Runtime coverage

The existing `XAIAutoExecutor` wrappers must route:

- generic `HttpRequest` calls used by management tooling;
- non-streaming HTTP execution;
- SSE streaming execution;
- downstream/upstream WebSocket execution;
- OAuth refresh after an auth ID exists.

Derived proxy data is transient. Refresh results restore the original persisted
auth proxy and remove transient route attributes.

Resin transport-level network failures are retried once with the same selected
xAI auth and the same derived Resin Account when replay is safe and no response
payload has been exposed downstream. Resin synchronously invalidates the failed
lease, so this second connection is allocated to a replacement node. If that
single replay also fails, the final failure is request-scoped across HTTP,
stream, WebSocket, and refresh paths. These failures must not cool a valid xAI
credential, retry through unrelated credentials, invoke EgressProxyPool, or
fall back to the CPA global proxy.

The replay contract is intentionally narrow:

- non-streaming executor requests and refreshes may replay once after a network
  error;
- generic HTTP requests replay only when the request body is replayable;
- streaming and WebSocket requests replay only before their first non-empty
  payload; mid-response failures remain request-scoped and are not replayed;
- upstream HTTP statuses, exact spending-limit 402 responses, configuration
  failures, caller cancellation, and client request errors are not replayed.

When exact spending-limit 402 lease rotation is enabled, its configured retry
budget remains separate from the single network replay. A network failure on
any initial or post-rotation attempt may consume at most one network replay for
that request; a 402 retry must rotate the deterministic Resin Account lease
through the authenticated admin API before rebuilding the route. Neither path
may select another xAI auth or fall back to another proxy backend.

## Error behavior

Enabled but unusable Resin configuration returns a request-scoped 503-style
error and does not fall back to CPA's global proxy. Error text must identify the
configuration class without including secret contents or credentialed URLs.

## Operational limitation

Resin's standard HTTP CONNECT tunnel cannot inspect encrypted xAI HTTP response
codes. This change does not release Resin leases or retry another Resin Account
when xAI returns an exact spending-limit 402.

## Production rollout contract

- Audit the live CPA and Resin stacks without printing secret values.
- Reuse the live Resin proxy token for CPA and generate a new CPA-only identity
  key with mode `0600`.
- Back up every changed production file before replacement.
- Keep Resin on `vps-gateway`; mount both CPA secret files read-only.
- Deploy reviewed Resin and CPA commits only through their existing GitHub
  Actions build-and-deploy workflows, without changing Resin subscriptions,
  Platforms, or existing auth files.
- Production containers must run immutable GHCR `sha-<full commit>` tags. Local
  `resin:*` and `cliproxyapi:*` tags are allowed only as rollback artifacts and
  must not remain the steady-state Compose image.
- A workflow-provided `CLI_PROXY_IMAGE` must take precedence over the server
  `.env`; `.env` remains the fallback only when no explicit image is supplied.
- Automated deploy regression coverage must prove both explicit-image
  precedence and `.env` fallback without invoking Docker or SSH.
- Verify container health, Resin forward proxy authentication, CPA startup,
  dynamic xAI Account creation, and one real CPA xAI request.
- Verify a production Resin node failure opens the node circuit at failure count
  one, rotates the same Account lease, and is recovered by CPA's single internal
  replay without returning the first proxy 502 downstream.
- Wait for Resin's workflow deployment and health gate before pushing CPA, then
  wait for CPA's workflow deployment and verify both running image names and
  OCI revision labels match the pushed commits.
- On any failed verification, restore the previous CPA files/image/config and
  leave Resin data intact.
