# Proposal: keep xAI Resin bootstrap connections alive

## Motivation

CPA must inspect the first meaningful xAI stream event before it can safely
decide whether an exact 402 or a pre-response network failure may be retried.
That safety property currently blocks the downstream handler long enough for
Cloudflare to return 524 when xAI first-token latency exceeds roughly two
minutes.

## Change

Run the already configured SSE keep-alive during the blocking stream setup
window. The keep-alive is a downstream-only SSE comment and is not a meaningful
model payload, so Resin's exact-402 and network retry decisions remain
unchanged. Stop and join the bootstrap writer before normal stream writes begin.

Make the production Grok inspection timer state explicit through
`ENABLE_GROK_INSPECTION_TIMER`. Deployment continues to install updated unit
files, but enables or disables the timer according to that value instead of
always enabling it.

## Compatibility and safety

- No public request or response schema changes.
- No timeout is added after an upstream connection is established.
- Fast startup failures still use their original HTTP status if no heartbeat
  has committed the response.
- Slow startup failures after a heartbeat use the existing protocol-compatible
  SSE terminal error shape.
- Resin Account identity, lease rotation, retry budgets, and explicit-proxy
  precedence are unchanged.
- The timer switch defaults to `true` for compatibility; production explicitly
  opts out with `false`.

## Deployment

Commit and push CPA only. GitHub Actions builds the immutable
`ghcr.io/dovelanlan/cliproxyapi:sha-<commit>` image and the production workflow
deploys it. Resin source and its workflow are not changed.
