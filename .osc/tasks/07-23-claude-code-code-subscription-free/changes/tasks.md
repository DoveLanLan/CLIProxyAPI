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

- [x] 4) Deploy code and update production runtime policy
  - Target: production image and `/opt/cliproxyapi/data/config.yaml`
  - Change: Deploy the verified build, enable cooldown persistence, and switch routing to round-robin with a backup and controlled restart/reload.
  - Verify: image revision, config values, container health, management API smoke test.

- [x] 5) Make quota inspection recoverable
  - Target: `/opt/cliproxyapi/scripts/run-grok-inspection.sh` and related timer behavior
  - Change: Remove rolling quota and transient probe failures from permanent-disable policy; retain low concurrency.
  - Verify: script configuration and a successful inspection run.

- [x] 6) Reinspect and remediate disabled xAI credentials
  - Target: Grok inspection results and management auth APIs on `bytevirt`
  - Change: Scan disabled accounts, re-enable freshly healthy credentials, quarantine quota/ambiguous results, back up and delete only hard-dead credentials.
  - Verify: aggregate before/after counts, backup manifest/checksum, no secrets in output.

- [x] 7) Complete repository and production quality gates
  - Target: Go packages, `cmd/server`, `.osc/quality-gate.md`, change closure docs
  - Change: Run required tests/build, check protected paths/security/rollback, and perform production regression monitoring.
  - Verify: all commands and results recorded; rollback artifacts present.

- [x] 8) Persist the permission-denied inspection policy
  - Target: `deploy/systemd/grok-inspection.service`, `deploy/systemd/grok-inspection.timer`, `deploy/scripts/remote-deploy.sh`, deployment docs
  - Change: Track and install the production units with `permission_denied` in the permanent-disable list while excluding quota/probe errors.
  - Verify: unit contents, shell syntax, compose render, server build, deployed unit environment, and timer state.

## Notes

- Production evidence before changes: 3435 xAI credentials, 441 active, 2994 disabled; 1169 recent log records contained free-usage exhaustion while `/v1/messages` remained HTTP 200.
- Focused executor/auth tests, affected package tests, `go test ./...`, and the required server build passed before deployment.
- The production inspection script is now tracked at `deploy/scripts/run-grok-inspection.sh`; its permanent class list excludes rolling quota and transient probe errors.
- Disabled-only production inspection completed at 2026-07-23T12:12:22+08:00: 1346 healthy/enable, 55 quota/keep, 92 probe-error/keep, 1023 permission-denied/keep, and 479 reauth/delete.
- The 479 hard-dead reauth files were backed up under a mode-0700 directory with mode-0600 artifacts and SHA-256 checksums, then deleted through the plugin apply API. Post-delete verification found zero target files remaining and 479 files in the backup archive.
- GitHub Actions deployed image revision `0672a88e4412aa2d3cc2c8697cdc963f0acc7a72`; the container remained running while the management API reported `disable-cooling: false`, `save-cooldown-status: true`, and `routing.strategy: round-robin`.
- The 1346 freshly healthy credentials were re-enabled after deployment. A follow-up inspection found two newly reauth credentials; both were backed up and deleted. Final verification found 1344/1344 remaining recovery targets enabled and none disabled.
- Permanent deletion backups contain 479 initial reauth files plus 2 follow-up reauth files (481 total), with restricted permissions and verified SHA-256 manifests. All 481 deleted source files were absent after apply.
- Final observed xAI aggregate was 3232 total, 1876 active, and 1356 disabled. The aggregate changed during remediation because credentials continued to be added independently.
- No `/v1/messages` traffic occurred in the final ten-minute observation window, so production `.cds` creation and client-visible retry reduction remain traffic-dependent observations rather than synthetic production tests.
- Follow-up production evidence on 2026-07-24 found 112 freshly classified `permission_denied` credentials still enabled because the untracked service override omitted that class. After backup, `daemon-reload`, and a manual safe apply, all 112 were disabled with no deletion; the durable repository change is task 8.
- The tracked service/timer passed Linux systemd verification and are installed by `remote-deploy.sh`; base and split-proxy compose renders, shell syntax, and the required server build passed.
