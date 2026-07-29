# Regression checklist

## Stream bootstrap

- [x] A delayed Claude-compatible stream emits `: keep-alive` before its first
      upstream event.
- [x] The temporary writer stops and joins before the normal handler resumes
      response writes.
- [x] Stopping before the configured interval leaves the response uncommitted.
- [x] Request cancellation stops the temporary writer without emitting bytes.
- [x] A startup error before the first heartbeat keeps its HTTP JSON status.
- [x] A startup error after a heartbeat keeps HTTP 200 and emits a Claude SSE
      `event: error` frame.
- [x] Focused race tests pass for the new helper and Claude bootstrap path.

## Resin and deployment

- [x] Existing Resin executor and helper tests pass without retry-budget or
      meaningful-payload changes.
- [x] Timer regression covers enabled, disabled, and invalid settings.
- [x] Invalid timer values invoke no systemd state-changing command.
- [x] Production `.env` contains only the intended persisted value
      `ENABLE_GROK_INSPECTION_TIMER=false` for this change.
- [x] Before deployment, the production timer/service remain disabled/inactive.
- [x] Immutable GitHub Actions image is deployed and verified on bytevirt.
- [x] Production timer/service remain disabled/inactive after deployment.
- [x] CPA and Resin production health checks pass after deployment.

## Repository gates

- [x] `gofmt -w .`
- [x] Focused and full Go tests
- [x] Required server build
- [x] Shell syntax and executable shell regressions
- [x] `git diff --check`
- [x] No changes under `internal/translator/**`
