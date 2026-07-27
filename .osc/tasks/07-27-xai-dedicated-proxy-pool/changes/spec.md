# Spec: xAI dedicated rotating proxy pool

## Scope

### In scope

- Config schema, normalization, examples, and hot-reload behavior.
- xAI HTTP, SSE, WebSocket, OAuth refresh, and executor `HttpRequest` routing.
- Stable lane assignment, rollout percentage, new-start rate limiting, exact
  402 A/B verification, IP/node quarantine, state persistence, and Mihomo API
  control.
- Management status and limited control endpoints.
- Optional unprivileged Mihomo Compose overlay and secret-free examples.
- Authenticated API-only subscription CRUD with write-only URLs, generated
  Mihomo configuration, transactional reload/rollback, and safe two-step delete.
- Focused unit/integration tests and OSC closure artifacts.

### Out of scope

- Production mutation, SSH, subscription secrets, TUN, transparent proxying,
  other provider routing, dashboard UI, notifications, and automatic lane
  scaling.
- Subscription web UI/CLI, arbitrary YAML, custom provider request headers,
  non-HTTPS subscriptions, and caller-controlled file paths.

## Acceptance Criteria

1. A missing or disabled `xai-proxy-pool` preserves the exact existing proxy
   priority and request behavior. (Config/proxy helper regression tests.)
2. An explicit auth `proxy_url` bypasses the pool. An enrolled xAI auth uses its
   lane URL directly and never the global proxy. (Executor tests.)
3. Rendezvous hashing is deterministic across restarts and adding a lane moves
   fewer auth IDs than modulo hashing. (Pool unit test.)
4. Rollout percentage is deterministic and allows staged legacy/pool routing.
   (Pool unit test.)
5. Each lane limits request starts to 30/minute with burst 3 by default; queue
   overflow is a request-scoped 503. Established streams receive no new read
   timeout. (Limiter/executor tests.)
6. Only HTTP 402 with code `personal-team-blocked:spending-limit`, before any
   downstream payload, triggers one alternate same-auth attempt. (HTTP, stream,
   and WebSocket-focused tests.)
7. Alternate success quarantines the original public IP for 24h, selects the
   verified node for the affected lane, persists state, and returns the retry
   result. A repeated exact 402 is returned as a credential failure. (Pool and
   executor tests.)
8. Missing/failed verification and exhausted pools fail closed with a
   request-scoped 503 plus Retry-After and do not fall back to global/direct.
   (Executor/auth-manager regression test.)
9. Pre-connect network failure rotates and retries only after a negative node
   check. Three mid-response failures within two minutes rotate future
   connections without replaying the current stream. Network nodes are
   quarantined for 10m, separate from IP quarantine. (Pool/stream tests.)
10. State reload preserves lane selections and unexpired quarantines without
    storing auth IDs, tokens, subscription URLs, or controller secrets.
    (Persistence test.)
11. Management API returns redacted lane/provider/counter state and supports
    refresh, rotate, check, quarantine, and unquarantine under existing
    management authentication. (Handler/route tests.)
12. The deployment overlay publishes no host ports, uses no TUN/privileged
    capabilities, starts from an intentionally empty fail-closed bootstrap, and
    supports multiple API-managed providers without containing live secrets.
    (Compose config/shell syntax/manual review.)
13. `internal/translator/**` remains unchanged; focused tests, `go test ./...`,
    and `go build -o test-output ./cmd/server` pass. (Quality gate.)
14. Subscription URLs enter only create/update request bodies and never appear
    in list/status/error/log output. Registry and generated config files are
    atomically persisted with mode `0600`. (Redaction/persistence tests.)
15. Enabled create/update succeeds only when Mihomo reloads the candidate,
    exposes the named provider, and reports at least one node. Any failure
    restores the prior running config and registry. (Pinned-Mihomo integration
    and controller transaction tests.)
16. The API registry is the sole subscription source. Disabled drafts may be
    stored without live validation; permanent delete requires disabled state and
    no active lane reference. (Handler/store tests.)
17. Concurrent mutations are serialized; callers cannot submit arbitrary YAML,
    paths, headers, userinfo URLs, fragments, local/private literals, or an
    unbounded number/size of sources. (Validation/concurrency tests.)

## Behavior / Requirements

### Configuration and precedence

- Top-level key: `xai-proxy-pool`.
- Default is disabled. Duration/rate defaults are normalized only for the pool.
- Effective proxy order is: explicit auth proxy, enrolled xAI pool lane, global
  proxy, context transport/direct legacy behavior.
- An enabled, fully rolled-out pool fails closed if no lane is ready.
- Explicit administrator disable restores legacy behavior.
- Lane count is the configured lane-list length; expansion is manual.

### Stable routing and rate limiting

- Compute auth-to-lane assignment with rendezvous hashing over stable auth ID
  and stable lane name. Do not mutate auth files or persist auth mappings.
- `rollout-percent` uses a deterministic auth-ID bucket. Non-enrolled auths use
  legacy behavior for staged rollout only.
- Limit new upstream starts per lane. Do not add response-body/read deadlines or
  terminate active streams during selection changes.

### Mihomo control and node admission

- Controller requests use a secret loaded from a file and a private base URL.
- Refresh and query all proxy providers independently. Continue using Mihomo's
  last provider cache when an update fails.
- Candidate nodes must be alive, have unique names, not be quarantined, and
  resolve to a public egress IP different from active/quarantined IPs.
- Prefer provider diversity and lower observed delay when multiple candidates
  qualify.
- Serialize probe-selector mutation and lane replacement to prevent stampedes.
- Recheck selected lane egress IPs on startup/provider changes and periodically.

### Failure classification

- Exact IP suspicion requires status 402 and the exact structured code
  `personal-team-blocked:spending-limit`.
- A/B retry uses the same auth and request through one verified alternate
  egress. It is allowed only before downstream payload.
- Alternate success: quarantine original egress IP for the configured IP
  duration, promote alternate node into the lane, and do not penalize auth.
- Alternate exact 402: return credential failure; do not quarantine either IP.
- Alternate unavailable/ambiguous: return request-scoped 503; do not quarantine
  the IP or permanently penalize auth.
- Quota 429, other 402s, 401/403, and mid-response errors never trigger IP A/B.
- Network failure state is node-scoped and shorter-lived than xAI IP quarantine.

### Persistence and observability

- Persist versioned JSON atomically with lane selection, selected egress IP,
  quarantine expiry/reason, failure windows, and aggregate counters.
- The operational pool state must not persist auth IDs, auth-to-lane mappings,
  controller secret, subscription URLs, proxy credentials, request bodies, or
  upstream errors. Subscription URLs may exist only in the dedicated mode-`0600`
  registry and generated Mihomo config required to consume them.
- Status may include configured lane/provider aliases, redacted proxy endpoints,
  public egress IPs, health, quarantine expiry, and aggregate counters.
- Logs use structured fields and redact proxy/controller URLs.

### Management API

- `GET /v0/management/xai-proxy-pool/status`
- `POST /v0/management/xai-proxy-pool/providers/refresh`
- `POST /v0/management/xai-proxy-pool/lanes/:lane/rotate`
- `POST /v0/management/xai-proxy-pool/lanes/:lane/check`
- `POST /v0/management/xai-proxy-pool/quarantine`
- `DELETE /v0/management/xai-proxy-pool/quarantine/:ip`
- Missing/disabled pool returns an operator-readable conflict/unavailable error.
- Existing status/control endpoints never read or write subscription URLs or
  arbitrary Mihomo config; only the dedicated POST/PUT subscription handlers
  accept write-only URL values.
- `GET /v0/management/xai-proxy-pool/subscriptions`
- `POST /v0/management/xai-proxy-pool/subscriptions`
- `PUT /v0/management/xai-proxy-pool/subscriptions/:name`
- `DELETE /v0/management/xai-proxy-pool/subscriptions/:name`
- `POST /v0/management/xai-proxy-pool/subscriptions/:name/check`
- GET/status/error output is redacted. POST/PUT accept a bounded write-only URL;
  disabled drafts can be persisted without activation.

### Subscription transaction

- CPA stores an independent versioned JSON registry and derives provider paths,
  prefixes, health checks, lane/probe groups, listeners, and rules itself.
- Provider names are stable bounded identifiers and cannot collide with lane,
  probe, or built-in names. URLs must use HTTPS, a public FQDN or public literal
  IP, a valid port, and no userinfo, fragment, or IPv6 zone; localhost and
  literal non-public IP targets are rejected.
- A single process mutex serializes registry mutations and Mihomo reload.
- Enabled candidate flow: render full YAML, reload candidate payload, refresh
  and query providers, require the target provider to contain at least one node,
  reconcile pool candidates/lanes, persist generated config and registry, then
  report success.
- Failure flow: reload the previous payload, restore prior in-memory registry,
  leave persisted files unchanged, and return a redacted request-scoped error.
- Disable flow removes the provider from candidate config, reloads, and rotates
  any affected lane before committing disabled state.
- Delete is rejected unless the provider is already disabled and no active lane
  references it; successful delete removes its registry entry and cached file.

## Edge Cases

- Nil auth, empty/duplicate lane names, invalid proxy/controller URLs, missing
  secret file, malformed controller JSON, empty providers, duplicate node names,
  duplicate egress IPs, IPv6 egress, and expired quarantine entries.
- Concurrent first-use initialization, simultaneous 402s on one lane, config
  reload during probe, canceled rate-limit waits, and state write failure.
- HTTP request bodies that cannot be replayed, HTTP 200 SSE error objects,
  WebSocket handshake errors, and errors after meaningful downstream payload.
- Existing xAI WebSocket session connected through an old route during lane
  rotation; it must complete and reconnect on the next request if needed.
- One subscription refresh failing while other cached providers remain usable.
- Candidate reload succeeds but provider download/parse fails; rollback reload
  fails; process restarts between runtime reload and persistence; duplicate
  provider names/fingerprints; disabled draft with unreachable URL; update of an
  in-use provider; last enabled provider disable; concurrent mutations; registry
  corruption; stale generated config; oversized body/provider download.

## Compatibility Notes

- No auth-file or database migration.
- Config is additive and disabled by default.
- SDK embedders receive the same xAI behavior when they enable the config.
- Public response formats remain handled by existing compatibility layers.
- The implementation must not touch `internal/translator/**`.
- Management API is additive and remains protected by existing middleware.
- Enabling subscription management requires new private registry/generated-file
  paths and a shared Mihomo directory mount. Existing static deployments remain
  supported when these paths are absent.

## API/UX Decisions

- Pool-local overload/unavailability uses HTTP 503 with Retry-After rather than a
  synthetic 402/429, so clients and auth scheduling do not confuse it with
  account state.
- Pool-local `Retry-After` is a CPA-generated managed header and remains visible
  when arbitrary upstream `passthrough-headers` is disabled.
- No dashboard is added; operators use Management API and structured logs.
- Node/provider names are operational metadata; URLs and secret contents are
  redacted or omitted.
