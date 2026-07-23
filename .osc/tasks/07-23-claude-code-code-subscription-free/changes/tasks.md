# Tasks: xAI streamed free-usage failover and Grok account-pool repair

- Date: 2026-07-23
- Owner(s): hewei
- Related: `spec.md`, `proposal.md`

## Assumptions

- Production remains `bytevirt` and uses `/opt/cliproxyapi/data/config.yaml`.
- The management plugin can inspect disabled xAI credentials without exposing their tokens.
- Permanent deletion will be based on fresh classifications, not existing disabled state.

## Checklist

- [x] 1) Add xAI HTTP SSE error classification
  - Target: `internal/runtime/executor/xai_executor.go`
  - Change: Detect explicit streamed error objects before protocol translation and reuse the existing free-usage 429 cooldown mapping.
  - Verify: Focused executor unit tests.

- [x] 2) Add end-to-end credential failover coverage
  - Target: `internal/runtime/executor/*_test.go`, `sdk/cliproxy/auth/*_test.go` as needed
  - Change: Prove a pre-payload streamed 429 cools the first auth and selects the next auth.
  - Verify: Focused package tests.

- [x] 3) Run local formatting and focused validation
  - Target: changed Go files
  - Change: Apply `gofmt` and fix all focused failures.
  - Verify: `go test` for executor/auth packages.

- [ ] 4) Deploy code and update production runtime policy
  - Target: production image and `/opt/cliproxyapi/data/config.yaml`
  - Change: Deploy the verified build, enable cooldown persistence, and switch routing to round-robin with a backup and controlled restart/reload.
  - Verify: image revision, config values, container health, management API smoke test.

- [ ] 5) Make quota inspection recoverable
  - Target: `/opt/cliproxyapi/scripts/run-grok-inspection.sh` and related timer behavior
  - Change: Remove rolling quota and transient probe failures from permanent-disable policy; retain low concurrency.
  - Verify: script configuration and a successful inspection run.

- [ ] 6) Reinspect and remediate disabled xAI credentials
  - Target: Grok inspection results and management auth APIs on `bytevirt`
  - Change: Scan disabled accounts, re-enable freshly healthy credentials, quarantine quota/ambiguous results, back up and delete only hard-dead credentials.
  - Verify: aggregate before/after counts, backup manifest/checksum, no secrets in output.

- [ ] 7) Complete repository and production quality gates
  - Target: Go packages, `cmd/server`, `.osc/quality-gate.md`, change closure docs
  - Change: Run required tests/build, check protected paths/security/rollback, and perform production regression monitoring.
  - Verify: all commands and results recorded; rollback artifacts present.

## Notes

- Production evidence before changes: 3435 xAI credentials, 441 active, 2994 disabled; 1169 recent log records contained free-usage exhaustion while `/v1/messages` remained HTTP 200.
- Focused executor/auth tests, affected package tests, `go test ./...`, and the required server build passed before deployment.
- The production inspection script is now tracked at `deploy/scripts/run-grok-inspection.sh`; its permanent class list excludes rolling quota and transient probe errors.
- Disabled-only production inspection completed at 2026-07-23T12:12:22+08:00: 1346 healthy/enable, 55 quota/keep, 92 probe-error/keep, 1023 permission-denied/keep, and 479 reauth/delete.
- The 479 hard-dead reauth files were backed up under a mode-0700 directory with mode-0600 artifacts and SHA-256 checksums, then deleted through the plugin apply API. Post-delete verification found zero target files remaining and 479 files in the backup archive.
- Healthy credentials remain disabled until the verified image, round-robin routing, and persisted cooldowns are active in production.
