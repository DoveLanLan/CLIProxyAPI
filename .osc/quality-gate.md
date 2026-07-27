# Quality Gate: xAI proxy pool and subscription Management API

Date: 2026-07-27
Task: `.osc/tasks/07-27-xai-dedicated-proxy-pool`
Status: PASS with pre-existing optional-vet findings

## Changed scope

- Additive xAI-only proxy-pool config and executor routing.
- Mihomo controller, lane/quarantine runtime, persistence, and management API.
- API-owned multi-subscription registry, generated configuration, hot-reload
  transaction, startup recovery, and two-step delete.
- Optional production Compose/deploy assets and operator documentation.
- No changes under `internal/translator/**`.

## Required and focused gates

| Command | Result |
|---|---|
| `gofmt` plus `gofmt -l` on all changed Go files | PASS |
| `go test ./internal/config ./internal/runtime/executor/helps ./internal/runtime/executor ./internal/api/handlers/management ./internal/api` | PASS |
| `go test -race ./internal/runtime/executor/helps ./internal/api/handlers/management` | PASS |
| `CLIPROXY_MIHOMO_INTEGRATION=1 go test -run TestXAIProxyGeneratedMihomoConfigPinnedImage ./internal/runtime/executor/helps` | PASS |
| `./osc gate --cmd "go test ./..."` | PASS |
| `go build -o test-output ./cmd/server && rm test-output` | PASS |
| `git diff --check` and trailing-whitespace scan of changed/untracked files | PASS |

The runtime used Go `1.26.0` on Darwin arm64.

## Deployment asset gates

| Check | Result |
|---|---|
| `bash -n deploy/scripts/remote-deploy.sh` | PASS |
| Base + xAI overlay `docker compose ... config --quiet` | PASS |
| Rendered Compose assertions: no Mihomo host ports/host network/privileged mode, `cap_drop: ALL`, controller secret read-only, shared config writable | PASS |
| Pinned Mihomo `-t` against `deploy/mihomo/config.example.yaml` | PASS |
| Pinned Mihomo `-t` against API-generated YAML | PASS |
| Local loopback-only Mihomo smoke: `PUT /configs?force=true` payload reload | PASS |

Pinned image:
`docker.io/metacubex/mihomo:v1.19.28@sha256:e6acd921addecfd59a8e2d38203f88356d635b54de6c0673db0e015139989312`

The smoke container was removed after validation. No production host, service,
configuration, or container was accessed or changed.

## Security and compatibility review

- PASS: subscription URL is accepted only by POST/PUT input and omitted from all
  response/status/error types.
- PASS: registry/generated files use atomic mode-`0600` writes; registry reads
  reject symlinks, unsafe permissions, oversized files, and invalid entries.
- PASS: provider names, HTTPS/FQDN/port URL shape, body size, provider count, URL
  length, and download size are bounded; userinfo, fragments, local names,
  non-public literal IPs, and IPv6 zones are rejected.
- PASS: candidate reload/refresh/verification/reconcile/persistence failures
  restore the prior runtime; startup reconstructs generated config from registry.
- PASS: mutations are serialized and protected by strict revision `If-Match`.
- PASS: established xAI streams receive no new read deadline and are not replayed
  or intentionally closed by subscription changes.
- PASS: explicit auth proxy remains higher priority and enrolled pool traffic
  never chains through global `proxy-url`.
- PASS: changed-file secret-pattern scan found no private key or common live-token
  pattern; examples use reserved placeholder domains/values.
- PASS: `git status --short internal/translator` is empty.

## Optional vet gate

`go vet ./...` reports six pre-existing findings and no new finding in the xAI
subscription-management files:

- `internal/logging/request_logger.go`: non-standard `WriteTo` signature.
- `sdk/api/handlers/handlers.go`: pre-existing unreachable statement at line
  1438 (outside this task's changed hunk).
- `internal/pluginhost/host_callbacks.go`: two cancel-path findings.
- `internal/pluginhost/host_model_stream_callbacks.go`: two cancel-path findings.

These findings are outside the task scope and remain as baseline technical debt.

## PR-ready checklist

- [x] Proposal, spec, tasks, summary, regression, and rollback artifacts updated.
- [x] Focused, race, integration, full-test, and required-build gates pass.
- [x] Deployment assets render and validate with the pinned Mihomo image.
- [x] Secret/redaction, rollback, compatibility, and protected-path reviews pass.
- [x] Production deployment remains separately authorized and was not performed.
