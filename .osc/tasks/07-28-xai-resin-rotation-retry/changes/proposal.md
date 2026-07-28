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

Merge this policy with the existing single pre-response Resin network retry.
The network and exact-402 budgets remain independent: network replay keeps the
current routed auth and Account, while exact 402 rotates only that Account's
lease before rebuilding the route.

Repair the CPA deploy entrypoint so an immutable GHCR image selected by GitHub
Actions cannot be overwritten by the VPS `.env`. Keep `.env` as the fallback
for invocations that do not supply an explicit image and cover both branches in
a shell test that does not invoke Docker or SSH.

## Compatibility

- The feature is disabled when `max-402-retries` is omitted or zero.
- Existing Resin proxy-only configuration remains valid without an admin token.
- Explicit per-auth proxies still bypass automatic Resin routing.
- EgressProxyPool behavior and the Resin/EgressProxyPool mutual-exclusion rule
  remain unchanged.
- Non-exact 402, 401, 403, 429, and mid-response failures retain their current
  behavior. Eligible pre-response network failures retain their one bounded
  same-route replay.
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

After local validation, push and deploy only through the existing GitHub
Actions workflows. Deploy Resin first and verify it, then deploy CPA and verify
that both VPS containers run the immutable GHCR `sha-<commit>` images selected
by their workflows before removing temporary local rollback tags.
