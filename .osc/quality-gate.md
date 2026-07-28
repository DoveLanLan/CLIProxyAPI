# Quality gate: xAI Resin retry and official deployment repair

Date: 2026-07-28
Tasks: `.osc/tasks/07-27-xai-resin-routing`, `.osc/tasks/07-28-xai-resin-rotation-retry`
Status: LOCAL PASS; OFFICIAL WORKFLOW DEPLOYMENT PENDING

## Changed scope

- Stable per-auth xAI Resin routing with no EgressProxyPool or global-proxy
  fallback while Resin is enabled.
- One same-auth/same-Account retry for pre-response Resin network failures.
- Bounded exact spending-limit 402 lease rotation through the Resin admin API.
- Independent network and exact-402 retry budgets across HTTP, SSE, WebSocket,
  management HTTP, and refresh paths.
- Production Resin overlay, config validation, secret redaction, and operator
  documentation.
- GitHub Actions image precedence over the server `.env`, with a sourceable
  resolver and Docker/SSH-free regression coverage.
- No changes under `internal/translator/**`.

## Required and focused gates

| Command | Result |
|---|---|
| `gofmt -w .` | PASS |
| `go test ./internal/runtime/executor -run 'Resin' -count=1` | PASS |
| `go test -race -run 'Resin' ./internal/runtime/executor -count=1` | PASS |
| `go test -race -run 'XAIResin' ./internal/runtime/executor/helps -count=1` | PASS |
| Focused `go vet` for all changed Go packages | PASS |
| `go test ./...` | PASS |
| `go build -o test-output ./cmd/server && rm test-output` | PASS |
| `bash -n` for deploy and image-resolution scripts | PASS |
| `deploy/scripts/resolve-cli-proxy-image_test.sh` | PASS |
| Base + Resin overlay `docker compose ... config` | PASS |
| `git diff --check` | PASS |
| Changed-path guard for `internal/translator/**` | PASS |

The release commit is ready for the required ordered workflow rollout: Resin
first, then CPA. Workflow run IDs, immutable image names, OCI revision labels,
health checks, and repeated production request results are verified after the
commit is pushed and therefore are reported with the deployment result rather
than predicted in this source artifact.

## Prior controlled production evidence

- CPA and Resin were healthy on `vps-gateway` using temporary rollback images.
- A scoped Resin node connection failure produced an internal `connect_dial`
  502, opened that node's circuit, and moved the same Account lease to another
  node.
- CPA's one pre-response retry hid the first proxy failure; both the management
  client and xAI upstream returned HTTP 200.
- Twenty consecutive real `grok-4.5` requests returned HTTP 200 with zero HTTP
  502 responses and a mean duration of 3.665 seconds.
- The CPA rollback backup remains at
  `/opt/cliproxyapi/backups/xai-resin-retry-20260728T121546Z`.

## Baseline notes

Full `go vet ./...` has existing warnings in unchanged logging, handler, and
plugin-host files. Focused vet covers all changed Go packages. `shellcheck` is
not installed; `bash -n` and the executable shell regression are required.
