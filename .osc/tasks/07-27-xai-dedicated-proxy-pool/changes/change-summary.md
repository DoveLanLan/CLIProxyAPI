# Change Summary: xAI dedicated rotating proxy pool

## Outcome

Implemented an opt-in xAI-only proxy pool backed by a private Mihomo sidecar.
The feature is disabled by default and does not alter legacy routing until an
operator enables it.

## Runtime behavior

- Routes enrolled xAI credentials to six configurable lanes with rendezvous
  hashing and deterministic rollout.
- Preserves explicit per-auth proxy precedence and prevents enrolled traffic
  from falling through to the global proxy.
- Limits new request starts per lane without adding response read deadlines.
- Emits CPA-managed `Retry-After` guidance for pool-local 503 responses even
  when arbitrary upstream-header passthrough is disabled.
- Uses one same-auth alternate egress attempt only for the exact HTTP 402
  `personal-team-blocked:spending-limit` condition before downstream payload.
- Separates 24-hour egress-IP quarantine from 10-minute network-node
  quarantine.
- Retries a pre-connect network failure only after Mihomo reports the node
  unhealthy; any second network failure remains request-scoped and does not
  penalize the credential.
- Observes but never replays mid-response failures. Three failures in the
  configured window rotate future connections.
- Includes the selected node in xAI WebSocket session identity so a lane
  rotation reconnects on the next request while the active stream is left
  alone.
- Persists lane selections, failure windows, counters, and quarantine state in
  atomic mode-`0600` JSON without auth IDs or secrets.
- Allows a failed startup initialization to recover through provider refresh
  without restarting the server.

## Operator surface

- Added redacted Management API status and controls for provider refresh, lane
  rotation/check, and IP quarantine/unquarantine.
- Routed xAI Management API inspection requests through the registered xAI
  executor so inspection receives the same lane and exact-402 behavior.
- Added an optional Compose overlay, a fail-closed empty Mihomo bootstrap, and a
  Chinese deployment guide. The sidecar publishes no host ports, uses no TUN or
  host networking, and drops all Linux capabilities.

## Subscription management extension

- Added authenticated list/create/update/check/delete Management API endpoints
  for multiple Mihomo subscription providers. Subscription URLs are write-only;
  responses expose only a safe host label, short fingerprint, node count,
  last-check time, and redacted error code.
- Added strong `If-Match` revision checks and response `ETag` headers so stale
  concurrent mutations cannot overwrite a newer registry.
- Added a versioned mode-`0600` registry and generated Mihomo configuration with
  bounded provider count, URL length, request body, and provider download size.
- Restricted inputs to bounded provider identifiers and HTTPS URLs without
  userinfo, fragments, localhost, or literal non-public IP targets.
- Enabled mutations now require a successful full reload, explicit provider
  refresh/download, at least one live node, and lane reconciliation before
  persistence. Candidate reload failures and later verification/persistence
  failures restore the prior running configuration.
- Startup reconstructs the generated Mihomo file from the registry and reloads
  that source of truth, closing the crash window between runtime activation and
  two-file persistence.
- Deletion remains two-step: disable and drain first, then permanently remove
  the registry URL and derived provider cache. Disabled drafts may be saved
  without contacting the subscription source.
- Updated the Compose overlay to share the private Mihomo directory with CPA and
  documented repeatable API workflows for multiple providers without routine
  SSH access.

## Compatibility

- Existing configs remain valid; the new top-level block defaults to disabled.
- No auth-file, database, or state migration is required.
- `internal/translator/**` was not changed.
- Production deployment and production configuration remain untouched and
  require separate authorization.
