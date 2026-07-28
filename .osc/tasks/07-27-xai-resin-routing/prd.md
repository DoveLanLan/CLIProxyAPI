# PRD: Add native Resin routing for xAI

## Problem

CLIProxyAPI can send xAI traffic through a static global or per-auth proxy URL,
but Resin needs a stable `Platform.Account` proxy identity to provide sticky
egress routing. Operators with hundreds or thousands of xAI credentials cannot
maintain one `proxy_url` value per auth file. Resin's HTTP CONNECT ingress also
cannot inspect the encrypted xAI `Authorization` header to derive the account.

## Goals

- Configure Resin once for all xAI credentials.
- Derive a stable, anonymous Resin Account from the selected CPA auth ID.
- Apply the derived proxy identity to xAI HTTP, SSE, WebSocket, management API
  calls, and post-login OAuth refresh without mutating persisted auth records.
- Keep explicit per-auth `proxy_url` as the highest-priority override.
- Keep the existing EgressProxyPool integration available, but prevent both
  automatic xAI egress backends from operating at the same time.
- Keep Resin proxy and identity secrets out of `config.yaml` and logs.

## Non-goals

- Reimplement Resin's node pool, subscriptions, health checks, or lease store.
- Make Resin compatible with the EgressProxyPool private API.
- Automatically release a Resin lease on xAI application-level 402 responses.
- Change non-xAI provider proxy routing.

## Acceptance criteria

- One `xai-resin-proxy` block routes every non-overridden xAI credential through
  Resin using a deterministic `Platform.xai-<digest>` username.
- The digest is HMAC-SHA256 over the stable auth ID using a CPA-only identity key
  file; raw auth IDs and credentials are not sent to Resin.
- Proxy authentication uses a separate Resin proxy-token file.
- Missing/invalid enabled settings fail closed for affected xAI requests.
- Resin transport failures remain request-scoped and do not cool xAI auths.
- Explicit auth `proxy_url` bypasses Resin.
- Disabled Resin preserves legacy/global-proxy behavior.
- HTTP/SSE/WebSocket and refresh wrapper paths are covered by focused tests.
- Config examples and deployment documentation explain both CPA and Resin setup.
- `gofmt -w .`, focused tests, `go test ./...`, and the required server build pass.
