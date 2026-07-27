# PRD: Add xAI dedicated rotating proxy pool

## Problem

High-volume Grok Free traffic from one VPS can cause xAI to reject otherwise
healthy credentials with HTTP 402 and
`personal-team-blocked:spending-limit`. A controlled A/B test confirmed that the
same credentials recover immediately when only the public egress IP changes.
The existing global/per-auth proxy selection cannot keep credentials stable on
multiple egress lanes, distinguish account failures from IP failures, or rotate
only the affected lane.

## Goals

- Add an opt-in xAI-only proxy pool without changing legacy `proxy-url` behavior.
- Bind each xAI auth ID deterministically to one of six configurable lanes.
- Verify the exact IP-block 402 through one alternate egress before penalizing
  the credential.
- Persist lane and quarantine state, rate-limit new connection starts, and fail
  closed when the dedicated pool is unavailable.
- Control Mihomo through its private API and expose redacted operational state
  through the existing Management API.
- Provide secret-free Docker deployment examples for multiple subscriptions.

## Non-goals

- TUN mode, host-wide proxying, TLS interception, or proxying non-xAI providers.
- Automatic lane-count expansion.
- A browser dashboard or external notification integration.
- Committing subscription URLs, proxy credentials, controller secrets, or live
  production configuration.
- Production installation or restart as part of this task.

## Acceptance criteria

- The feature is disabled by default and legacy proxy priority is unchanged.
- Explicit auth `proxy_url` overrides the xAI pool; selected pool traffic never
  chains through the global `proxy-url`.
- Stable rendezvous hashing maps auth IDs to configured lanes and minimizes
  movement when a lane is added.
- Only the exact pre-response 402 code triggers one same-credential alternate
  retry; successful A/B quarantines the original egress IP for 24 hours, while
  a second exact 402 remains a credential error.
- Pool exhaustion and unverifiable A/B failures return request-scoped 503 errors
  without permanently disabling the credential or falling back to another proxy.
- Existing streams are not closed when a lane rotates, and mid-response failures
  are never replayed.
- Focused tests, `go test ./...`, and the required server build pass.

## Extension request: subscription management API

### Problem

The initial deployment design keeps subscription URLs in a root-only
`mihomo.yaml`, so adding or replacing a provider currently requires an operator
to connect to the VPS and edit that file. The user wants a fast Management API
workflow for adding, replacing, disabling, testing, and deleting subscription
sources without routine SSH access.

### Goal

- Add authenticated CRUD endpoints under the existing private Management API.
- Apply subscription changes through Mihomo configuration reload without a CPA
  or Mihomo container restart.
- Make each mutation transactional: validate, write atomically, reload, verify,
  and restore the previous file/configuration if activation fails.
- Return provider status and node counts without exposing subscription tokens.

### Known facts

- The current Compose overlay mounts `mihomo.yaml` read-only and does not expose
  it to CPA, so the volume layout must change before CPA can own mutations.
- Mihomo officially supports full configuration reload through
  `PUT /configs?force=true` and individual provider refresh through
  `PUT /providers/proxies/:name`.
- Mihomo restricts reload paths to its working directory unless `SAFE_PATHS` is
  configured.
- The existing Management API is already authenticated and production access is
  intended to stay private/Tailscale-only.

### Recommended technical direction

- Store API-managed sources in a dedicated mode-`0600` JSON registry and render
  a generated Mihomo YAML file into a CPA/Mihomo shared private volume.
- Keep the fixed listener, selector, health-check, and security topology under
  application control; accept only provider name, HTTPS subscription URL, and
  enabled state from the API.
- On create/update/delete: lock mutations, render a candidate, atomically write,
  request Mihomo reload, inspect the provider snapshot, and roll back both file
  and running config on failure.
- Never log request bodies or URL values. Identify URLs in responses by a short
  fingerprint and optionally a redacted host only.
- Suggested endpoints:
  - `GET /v0/management/xai-proxy-pool/subscriptions`
  - `POST /v0/management/xai-proxy-pool/subscriptions`
  - `PUT /v0/management/xai-proxy-pool/subscriptions/:name`
  - `DELETE /v0/management/xai-proxy-pool/subscriptions/:name`
  - `POST /v0/management/xai-proxy-pool/subscriptions/:name/check`

### Constraints

- No public unauthenticated endpoint and no direct Mihomo controller exposure.
- No subscription URL in CPA config, logs, status, Git, or backup diagnostics.
- No arbitrary YAML upload or arbitrary filesystem path supplied by callers.
- Mutations must be serialized and must not restart or terminate established
  xAI streams.
- Production installation remains separately authorized.

### Open decisions

- None for the API-only MVP.

### Confirmed extension decisions

- Subscription URLs are write-only. No GET/status/error response returns the
  original URL; responses expose only non-secret provider metadata, a redacted
  host when safe, and a short fingerprint.
- Once subscription management is enabled, the API registry is the sole source
  of truth for proxy-provider subscriptions. Existing sources are imported once
  through the API; later manual edits to generated provider configuration are
  unsupported and must not be merged back implicitly.
- Provider deletion is a two-step operation. The provider must first be disabled,
  the generated config reloaded, and affected lanes migrated away. Permanent
  `DELETE` is accepted only after no active lane references that provider; it
  then removes the write-only URL and provider cache metadata.
- The first version exposes the authenticated Management API only. A command-line
  helper and CPA Manager Plus web UI are deferred until production behavior has
  been validated.
- Enabling a new or updated subscription is a strict transaction: the source
  must download, parse into at least one valid node, reload successfully, and
  appear in the controller snapshot before the registry update is committed.
  Any failure restores the previous file and running configuration. Disabled
  sources may be saved as unvalidated drafts.

### API contract for the MVP

- `GET /v0/management/xai-proxy-pool/subscriptions`
  - Returns name, enabled state, safe hostname label, URL fingerprint, node
    count, last check time, and redacted error state.
  - Never returns a URL path, query, fragment, credentials, or raw controller
    error containing the URL.
- `POST /v0/management/xai-proxy-pool/subscriptions`
  - Body: stable provider `name`, write-only `url`, and optional `enabled`.
  - Creates a provider or returns conflict when the name already exists.
- `PUT /v0/management/xai-proxy-pool/subscriptions/:name`
  - Replaces the write-only URL and/or changes enabled state.
  - Enabling follows the strict activation transaction.
- `DELETE /v0/management/xai-proxy-pool/subscriptions/:name`
  - Rejects enabled/in-use providers. Permanently deletes only a disabled,
    drained provider.
- `POST /v0/management/xai-proxy-pool/subscriptions/:name/check`
  - Refreshes and health-checks one enabled provider without returning its URL.

### Validation and limits

- Provider names use a bounded stable identifier and cannot collide with lane,
  probe, or Mihomo built-in names.
- Subscription URLs are bounded HTTPS URLs with a public FQDN or public literal
  IP, valid ports, and no userinfo, fragments, or IPv6 zones.
- Localhost, loopback, link-local, and literal private-address targets are
  rejected; callers cannot submit headers, filesystem paths, or arbitrary YAML.
- Provider count, request body size, downloaded subscription size, and
  concurrent mutations are bounded.
- Mutations are serialized. Each response reports whether persistence, reload,
  provider discovery, and lane reconciliation succeeded.

### Extension acceptance criteria

- An authenticated caller can create, list, update, disable, check, and safely
  delete multiple subscription sources without SSH or container restart.
- Full subscription URLs are accepted only in write requests and never appear
  in API responses, logs, state/status output, or OSC diagnostics.
- API-managed registry data and generated Mihomo configuration use mode `0600`
  atomic persistence and are excluded from Git.
- Successful enabled mutations are visible in the Mihomo provider snapshot and
  expose at least one valid node before commit.
- Failed download, parse, reload, discovery, persistence, or reconciliation
  restores the previous registry and running Mihomo configuration.
- Concurrent or stale mutations cannot overwrite a newer committed registry.
- Disabling drains lanes from the provider before permanent deletion is allowed.
- Existing xAI streams receive no new read timeout and are not replayed or
  intentionally terminated by subscription mutation.
- Focused transaction/redaction/concurrency tests, Mihomo integration tests,
  `go test ./...`, and the required server build pass.

### Extension out of scope

- CPA Manager Plus UI, browser forms, and a standalone CLI helper.
- Public Management API exposure or a second authentication system.
- Arbitrary Mihomo YAML editing, custom provider headers, non-HTTPS source URLs,
  or caller-controlled storage paths.
- Automatic purchasing, subscription discovery, or automatic provider-count
  scaling.
- Production installation or migration without separate authorization.
