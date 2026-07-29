# Spec: bootstrap SSE keep-alive and durable timer disable

## Bootstrap keep-alive contract

For Claude-compatible streaming requests, start a temporary keep-alive writer
immediately before the synchronous stream setup call.

- Use `streaming.keepalive-seconds` as the interval.
- When disabled or when the response writer cannot flush, do nothing.
- On each tick, set the normal SSE headers, write `: keep-alive\n\n`, and flush.
- Stop and join the writer before the request handler resumes response writes.
- Return whether a heartbeat was written so error handling knows whether HTTP
  headers were committed.
- Respect downstream cancellation and never keep a goroutine alive after stop.

The heartbeat is not passed into executor chunk accounting and therefore does
not consume the "first meaningful payload" boundary used by Resin exact-402,
network, or generic bootstrap retries.

If stream setup fails before a heartbeat, preserve the existing JSON HTTP error
response. If it fails after a heartbeat, keep HTTP 200 and emit the existing
Claude-compatible SSE `event: error` frame.

## Timer deployment contract

Add `ENABLE_GROK_INSPECTION_TIMER` to the deployment environment.

- Accepted values: `true` and `false` only.
- Default: `true`, preserving historical installations.
- Always install both tracked systemd unit files and run `daemon-reload`.
- `true`: `systemctl enable --now grok-inspection.timer`.
- `false`: `systemctl disable --now grok-inspection.timer` and stop any active
  `grok-inspection.service`.
- Invalid values fail before changing timer state.

Production `/opt/cliproxyapi/.env` must contain
`ENABLE_GROK_INSPECTION_TIMER=false` before the new image deployment.

## Verification

- Unit tests cover bootstrap heartbeat emission, flush/stop behavior, and the
  no-heartbeat path.
- Claude handler tests cover terminal errors after a committed heartbeat.
- Shell tests cover both timer states and invalid input without calling real
  systemd.
- Existing Resin retry suites, all Go tests, shell syntax, and the required
  server build pass.
- Production shows the immutable commit revision, healthy CPA/Resin, a disabled
  and inactive timer/service, and downstream heartbeat bytes before a delayed
  first model event.
