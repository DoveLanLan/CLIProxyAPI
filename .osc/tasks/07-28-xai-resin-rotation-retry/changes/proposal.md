# Proposal: rotate Resin leases after exact xAI 402 responses

## Motivation

CPA currently assigns one stable Resin Account to each xAI credential. This is
desirable during normal traffic, but a public IP blocked by xAI remains attached
to the Account. The exact upstream response
`402 personal-team-blocked:spending-limit` has been verified by the operator to
clear when the same credential is retried through a different public IP.

## Proposed change

Extend `xai-resin-proxy` with optional Resin admin API settings and a bounded
`max-402-retries` policy. After the exact 402, CPA deletes only the current
Account lease through Resin's authenticated admin API and retries the same xAI
credential. Resin recreates the stable Account lease on the next proxy
connection and selects an available route.

Retries are permitted only before CPA has emitted a downstream payload. Generic
HTTP calls are retried only when their bodies are replayable. Streaming and
WebSocket traffic may retry only during bootstrap or handshake and is never
replayed mid-response.

## Compatibility

- The feature is disabled when `max-402-retries` is omitted or zero.
- Existing Resin proxy-only configuration remains valid without an admin token.
- Explicit per-auth proxies still bypass automatic Resin routing.
- EgressProxyPool behavior and the Resin/EgressProxyPool mutual-exclusion rule
  remain unchanged.
- Non-exact 402, 401, 403, 429, network errors, and mid-response failures retain
  their current behavior.
- No public route, payload schema, auth-file, or translator change is required.

## Security

- The Resin admin token is read from a file and never placed directly in YAML.
- Admin requests stay on the configured internal Resin URL and use Bearer auth.
- Logs and public errors must not contain proxy/admin tokens, credentialed proxy
  URLs, raw auth IDs, derived Account names, or upstream credentials.
- Lease deletion is scoped to the deterministic Account derived for the
  currently selected xAI credential.

## Rollback

Set `max-402-retries: 0` or remove the admin settings. This restores the current
single-attempt Resin behavior without changing auth files or existing leases.
The full feature can be rolled back by reverting this change and removing the
admin-token secret mount.

## Deployment boundary

This change package authorizes local repository changes and tests only. VPS
configuration, secret creation, container replacement, or service restart
requires a separate explicit production authorization after local validation.
