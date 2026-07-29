# Quality gate: xAI Resin stream bootstrap keep-alive

Date: 2026-07-29
Task: `.osc/tasks/07-29-xai-resin-bootstrap-keepalive`
Status: PASS; OFFICIAL WORKFLOW DEPLOYMENT VERIFIED

## Changed scope

- Downstream SSE heartbeats during the blocking Claude-compatible stream
  bootstrap wait.
- Claude SSE terminal-error handling after a bootstrap heartbeat commits 200.
- Explicit production Grok inspection timer deployment state.
- Deployment template and operator documentation.
- No Resin source changes and no changes under `internal/translator/**`.

## Required and focused gates

| Command | Result |
|---|---|
| `gofmt -w .` | PASS |
| `go test -race ./sdk/api/handlers -run StreamingBootstrapKeepAlive -count=5` | PASS |
| `go test -race ./sdk/api/handlers/claude -run 'Bootstrap\\|StartupError' -count=1` | PASS |
| `go test ./sdk/api/handlers/... ./internal/runtime/executor/...` | PASS |
| `go test ./internal/runtime/executor -run Resin -count=1` | PASS |
| `go test ./...` | PASS |
| `go build -o test-output ./cmd/server && rm test-output` | PASS |
| `bash -n deploy/scripts/*.sh` | PASS |
| `deploy/scripts/resolve-cli-proxy-image_test.sh` | PASS |
| `deploy/scripts/configure-grok-inspection-timer_test.sh` | PASS |
| `git diff --check` | PASS |
| Changed-path guard for `internal/translator/**` | PASS |

## Production preparation

- bytevirt `/opt/cliproxyapi/.env` now contains
  `ENABLE_GROK_INSPECTION_TIMER=false`; no other environment values were read
  back or recorded.
- Before deployment, `grok-inspection.timer` is disabled/inactive and
  `grok-inspection.service` is inactive.

## Production verification

- Commit: `d6027f472377d9e05af927700b57058fd9c67483`
- Docker workflow: `docker-image` run `30416414868`, success.
- Deployment workflow: `deploy-production` run `30416499287`, success.
- Image: `ghcr.io/dovelanlan/cliproxyapi:sha-d6027f472377d9e05af927700b57058fd9c67483`
- OCI revision matches the commit; the CPA container is running and `/healthz`
  returns 200 through both the private binding and Cloudflare public route.
- Resin is running with Docker health `healthy`; its `/healthz` returns 200.
- After deployment, `grok-inspection.timer` remains disabled/inactive and
  `grok-inspection.service` remains inactive.
- A real Cloudflare-facing `grok-4.5` Claude SSE request returned HTTP 200 in
  49.25 seconds. The first SSE item was a bootstrap heartbeat, three heartbeats
  arrived before the first model payload, and the payload then completed
  normally.

## Baseline note

A broad `go test -race ./sdk/api/handlers` run reports an existing race in
`TestHandlerStreamInterceptorInitializesHeadersBeforeReturn` between the test's
header read and the existing stream interceptor's header replacement. The race
does not involve the new response writer or changed files. Focused race runs for
all new bootstrap tests pass. `shellcheck` is not installed; shell syntax and
executable regression tests pass.
