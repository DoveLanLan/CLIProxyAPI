# Change summary

## CPA runtime

- Added a joined SSE bootstrap heartbeat writer using the existing
  `streaming.keepalive-seconds` setting.
- Wired it around the blocking Claude-compatible stream setup call used by the
  observed `/v1/messages` production route.
- Preserved normal HTTP startup errors when no heartbeat was emitted.
- Encoded later startup failures as Claude SSE terminal errors after a heartbeat
  committed HTTP 200.
- Left Resin retry, lease rotation, Account identity, payload classification,
  and all upstream timeout behavior unchanged.

## Deployment

- Added `ENABLE_GROK_INSPECTION_TIMER=true|false`, defaulting to `true`.
- Deployment always installs and reloads the tracked units.
- Enabled mode starts the timer; disabled mode disables/stops the timer and
  stops any active inspection service.
- Added shell regressions for enabled, disabled, and invalid settings.
- Persisted `ENABLE_GROK_INSPECTION_TIMER=false` on bytevirt before deployment.

## Delivery

Only CLIProxyAPI is changed. The immutable CPA image is built and deployed by
the existing GitHub Actions workflows; no local image or custom tag is used.
Resin source and its workflow are untouched.

Production is running
`ghcr.io/dovelanlan/cliproxyapi:sha-d6027f472377d9e05af927700b57058fd9c67483`
with a matching OCI revision. A real Cloudflare-facing `grok-4.5` stream
returned HTTP 200 after 49.25 seconds, with three bootstrap heartbeats before
the first model payload. CPA and Resin health checks pass, while the Grok timer
and service remain disabled/inactive.
