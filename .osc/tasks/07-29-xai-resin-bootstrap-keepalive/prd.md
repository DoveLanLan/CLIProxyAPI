# Bugfix: Prevent Cloudflare 524 during xAI Resin bootstrap

## Problem

Long-running `grok-4.5` requests sent through the Claude-compatible SSE route
can spend more than Cloudflare's approximately 120-second origin response
window waiting for the first meaningful upstream event. The production
configuration requests a 15-second streaming keep-alive, but the keep-alive
forwarder starts only after synchronous Resin bootstrap inspection returns.

Production evidence on 2026-07-29 showed an origin-side Nginx 499 after 123.6
seconds, a CPA `context canceled` response, and a healthy Resin CONNECT tunnel.

The CPA production deploy script also unconditionally enables the five-minute
Grok inspection timer, undoing an operator-requested disable on every deploy.

## Reproduction

1. Send a large `stream: true` `/v1/messages` request for `grok-4.5` through
   Cloudflare and Resin.
2. Let xAI take more than 120 seconds to emit the first SSE event.
3. Observe Cloudflare 524, Nginx 499, and CPA cancellation despite
   `streaming.keepalive-seconds: 15`.
4. Run the production deploy script after disabling `grok-inspection.timer` and
   observe that the script enables it again.

## Expected behavior

- CPA emits SSE comment heartbeats during the synchronous bootstrap wait.
- Exact-402 and pre-response network retries remain eligible until a meaningful
  model payload is delivered.
- Immediate failures that happen before the first heartbeat retain their normal
  HTTP error status.
- Deployments respect an explicit production timer switch and do not re-enable
  a deliberately disabled timer.

## Root cause

The handler starts normal streaming keep-alives only after
`ExecuteStreamWithAuthManager` returns. Resin bootstrap synchronously reads the
first chunk so it can safely classify and replay exact pre-response failures.
No bytes therefore reach Cloudflare during a slow first-token wait.

`remote-deploy.sh` also executes `systemctl enable --now` without consulting a
deployment setting.

## Fix

- Start a bounded SSE bootstrap keep-alive writer before the blocking stream
  setup call and stop/join it before normal response writes resume.
- Use the existing configured streaming keep-alive interval.
- If a heartbeat committed HTTP 200, surface a later startup failure as a
  protocol-compatible SSE error event.
- Add `ENABLE_GROK_INSPECTION_TIMER`, defaulting to the historical enabled
  behavior, and set production to `false`.
- Install the systemd units on every deploy but enable or disable them according
  to the explicit switch.

## Regression tests

- [x] Bootstrap waits emit and flush SSE comment heartbeats.
- [x] Stopping before the interval emits no bytes and preserves HTTP error status.
- [x] Existing exact-402, network retry, and post-payload no-replay tests pass.
- [x] Timer helper covers enabled, disabled, and invalid settings.
- [x] Focused tests, full tests, and required server build pass.
