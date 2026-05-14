# Tasks: Integrate CPA-Manager Panel and Usage Monitoring

- Date: 2026-05-14
- Owner(s): hewei
- Related: proposal.md, spec.md

## Assumptions

- Use the current branch and backport only the minimal CPA-Manager compatibility surface.
- Use `seakee/cpa-manager:latest` as the external Usage Service image.
- Do not implement RESP queue compatibility in this change.

## Checklist

- [x] 1) Add usage queue core
  - Target: `internal/redisqueue`, `sdk/cliproxy/usage`, runtime usage helpers
  - Change: queue usage payloads with auth/model/token/latency/failure metadata, retention, and config toggles.
  - Verify: queue/plugin unit tests.

- [x] 2) Expose management compatibility APIs
  - Target: `internal/api/handlers/management`, `internal/api/server.go`, auth runtime stats
  - Change: add `usage-queue` and `api-key-usage` endpoints under existing management auth.
  - Verify: handler tests.

- [x] 3) Switch management panel source
  - Target: `internal/config`, `config.example.yaml`, `internal/managementasset`
  - Change: default panel repository and fallback URL point to CPA-Manager.
  - Verify: build and code review.

- [x] 4) Update deployment examples
  - Target: compose files and deploy docs
  - Change: add optional CPA-Manager Usage Service with persistent `/data` volume and private-access guidance.
  - Verify: file review.

- [x] 5) Run gates
  - Target: whole repo
  - Change: gofmt and quality checks.
  - Verify: `go test ./...` and `go build -o test-output ./cmd/server && rm test-output`.

## Notes

- CPA-Manager release `v1.2.1` publishes `management.html`; the existing updater asset name matches it.
