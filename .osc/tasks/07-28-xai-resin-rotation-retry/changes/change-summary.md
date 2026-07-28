# Change summary: bounded xAI retry through Resin lease rotation

## Outcome

CPA now keeps the existing stable Resin Account for each xAI credential during
healthy traffic and can recover from the exact verified IP-block response
`402 personal-team-blocked:spending-limit`. When configured, CPA deletes only
that Account's Resin lease through the authenticated internal admin API and
retries the same credential through a fresh connection. The recommended setting
is two additional attempts, for at most three total upstream requests.

## Runtime behavior

- `max-402-retries` defaults to zero and is bounded to 0–5.
- Positive retry counts require a valid Resin admin URL and token file; invalid
  settings fail closed with a request-scoped 503.
- Platform IDs are resolved by exact name, cached, and refreshed after a stale
  404. Lease deletion accepts 204 and an already-absent 404.
- Concurrent 402 bursts for one Account are coalesced by observed lease
  generation so they do not repeatedly delete a newly created lease.
- Non-stream execution and OAuth refresh retry only the exact status/code pair.
- Generic HTTP requests retry only when the body can be recreated with
  `GetBody`; rejected responses are closed before replay.
- SSE retries only before the first non-empty payload. WebSocket retries only
  during handshake/bootstrap. Mid-response failures are never replayed.
- A per-auth lease generation changes the WebSocket session target after
  rotation, preventing reuse of the old tunnel without proactively terminating
  unrelated in-flight streams.
- Exhausted exact-402 retries remain status-compatible but are request-scoped,
  so the conductor does not cool or disable the credential for an IP failure.
- The one pre-response Resin network retry has an independent request budget.
  It keeps the current routed auth and Account before or after a 402 rotation;
  combined orderings are covered by non-stream and stream regressions.
- Explicit auth proxies still bypass Resin, and Resin never falls back to
  EgressProxyPool or CPA's global proxy.

## Security and compatibility

- The admin token is file-backed and sent only as a Bearer header to the
  configured direct HTTP(S) Resin admin endpoint.
- Errors and config diffs do not expose tokens, raw auth IDs, derived Account
  names, credentialed URLs, admin response bodies, or secret file paths.
- Existing Resin configurations with zero/omitted retries keep their previous
  single-attempt stream behavior and do not require admin credentials.
- xAI OAuth token errors now preserve HTTP status and response body through a
  typed internal error so the exact 402 classifier also works during refresh.
- No public API schema, auth-file layout, database, or translator changed.

## Deployment assets

The Resin Compose overlay supports a third read-only admin-token mount. The
deployment script validates retry count and requires the admin token only when
the deployment-side retry value is positive. The environment template remains
at zero retries by default. `config.example.yaml` and the Chinese setup guide
document the recommended value, secret preparation, behavior, verification,
limitations, and rollback.

The deployment entrypoint now preserves an immutable `CLI_PROXY_IMAGE` supplied
by GitHub Actions over the server `.env`, while retaining `.env` as the fallback
for invocation without an explicit image. A sourceable resolver and standalone
shell regression cover both branches without Docker or SSH.

The official rollout is performed only after this commit, through Resin's
workflow first and CPA's workflow second. No manual image build, transfer, or
custom steady-state tag is part of this delivery.
