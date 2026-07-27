# Tech notes

## Architecture decisions

- One unprivileged Mihomo Docker sidecar provides multiple HTTP listeners and
  selector groups; CPA controls only the private Mihomo API.
- Six active lanes plus one probe selector are the initial production topology.
- Auth-to-lane mapping uses rendezvous hashing and is computed from stable auth
  IDs; auth files are not rewritten.
- Multiple subscription URLs remain in a root-only production file and become
  separate Mihomo proxy providers.
- The exact HTTP 402 `personal-team-blocked:spending-limit` is treated as an
  egress suspicion until one same-auth A/B retry confirms whether the failure
  follows the credential or the IP.
- IP quarantine (24h) and network-node quarantine (10m) are separate states.
- New connection starts are limited to 30/minute/lane with burst 3; established
  SSE/WebSocket connections have no added read timeout.

## Risks / mitigations

- Shared nodes can already be blocked: discover actual egress IP before
  admission, deduplicate it, and use real 402 feedback for xAI blocking.
- Concurrent 402s can stampede rotation: serialize probe selection and lane
  replacement.
- Proxy-pool failures can poison auth cooldowns: return request-scoped errors
  for pool-local failures.
- WebSocket reuse can retain an old route: include the effective proxy route in
  xAI session target identity while leaving active connections untouched.
- Secrets can leak through config/logs: store only secret-file paths in CPA and
  redact controller/proxy URLs in logs and status.

## Rollback plan

- Keep the feature disabled by default.
- An explicit administrator disable restores legacy auth proxy -> global proxy
  -> direct behavior; automatic pool failure remains fail-closed.
- Remove the optional Compose overlay to remove Mihomo without affecting the
  base production stack.
- Revert the feature commit; the state file contains no credentials or secrets
  and may be archived or deleted independently.
