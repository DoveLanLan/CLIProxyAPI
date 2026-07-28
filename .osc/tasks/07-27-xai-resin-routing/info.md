# Tech notes

- Add a narrow xAI Resin routing helper under `internal/runtime/executor/helps`.
- Use HMAC-SHA256 with a separate stable identity key; use the first 128 bits as
  a lowercase hex account suffix.
- Construct proxy userinfo with `net/url`, never string concatenation.
- Priority: explicit auth proxy > Resin automatic route > EgressProxyPool route
  > global CPA proxy/context transport.
- If both automatic backends are enabled, Resin routing fails closed with a
  request-scoped service-unavailable error.
- Resin standard CONNECT cannot observe xAI's encrypted HTTP 402. Automatic
  lease release is deliberately outside this change.
- Rollback: disable/remove `xai-resin-proxy`, remove its secret mounts, and
  restart or hot-reload CPA. Persisted auth files are untouched.
