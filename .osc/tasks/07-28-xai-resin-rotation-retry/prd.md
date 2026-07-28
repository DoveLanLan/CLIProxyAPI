# PRD: Add xAI Resin request rotation and 402 retry

## Problem

The deployed CPA-to-Resin integration derives one stable Resin Account per xAI
credential. Resin therefore keeps that credential on a sticky egress lease.
When xAI returns the exact HTTP 402 code
`personal-team-blocked:spending-limit`, the current `resinUsed` executor path
returns the error after one attempt and does not invoke the older
EgressProxyPool A/B retry.

The operator wants Resin-backed xAI traffic to rotate egress more aggressively
and wants CPA to retry an exact pre-response 402 through a new Resin route.

## Goals

- Add a bounded, explicit Resin rotation policy for xAI requests.
- Retry only the exact spending-limit 402 and only before downstream payload.
- Cover non-stream HTTP, replayable Management API HTTP calls, SSE bootstrap,
  WebSocket handshake, and OAuth refresh paths.
- Keep retries request-scoped so an IP-related failure does not disable an xAI
  credential.
- Preserve explicit per-auth proxy precedence and fail-closed behavior.

## Non-goals

- Replaying a stream after meaningful payload was delivered.
- Retrying arbitrary 402, 401, 403, 429, or malformed upstream errors.
- Guaranteeing that Resin never selects the same public IP twice unless Resin
  exposes and enforces an exclusion contract.
- Mutating production before local code/tests and a separate deployment
  authorization.

## Acceptance criteria

- The selected rotation mode is documented and disabled or backward-compatible
  by default.
- Exact 402 retry count is configurable and strictly bounded.
- Each retry starts a new upstream connection through Resin.
- Non-replayable generic HTTP request bodies fail safely without a retry.
- SSE/WebSocket retries occur only during bootstrap/handshake, never midstream.
- Focused tests, full tests, and the required server build pass.

## Confirmed repository and Resin facts

- Current CPA always sends Resin V1 credentials as
  `<Platform>.<stable-account>:<proxy-token>`; this deliberately creates a
  sticky lease per selected xAI auth.
- Current CPA bypasses exact-402 handling whenever `resinUsed` is true.
- Resin's official non-sticky forward-proxy form omits Account and lets Resin
  choose a route per new connection.
- Resin also exposes authenticated APIs to list/delete a specific Account lease,
  but using them requires CPA to receive a Resin admin token and resolve the
  Platform ID.
- CPA constructs a fresh proxy-aware HTTP transport for routed xAI execution;
  WebSocket retries create a new dial. A non-sticky retry can therefore open a
  new Resin connection without adding a post-connect read timeout.

## Confirmed decision

- Keep the current stable Resin Account per xAI credential during healthy
  traffic. Do not switch to non-sticky per-request routing.
- When the exact pre-response spending-limit 402 is observed, delete that
  Account's Resin lease and retry with the same credential and Account so Resin
  assigns a fresh route.
- Configure the maximum number of additional exact-402 retries. The recommended
  deployment value is `2`, for at most three total upstream attempts.
- Preserve backward compatibility: omitted or zero retries keep the existing
  single-attempt behavior and do not require Resin admin credentials.
