# Regression Checklist: xAI streamed free-usage failover and Grok account-pool repair

- Date: 2026-07-23
- Related: `spec.md`, `tasks.md`

## Repository gates

- [x] Changed Go files formatted with `gofmt`.
- [x] Focused xAI executor tests passed.
- [x] Focused auth-manager failover tests passed.
- [x] Affected executor and auth package tests passed.
- [x] `go test ./...` passed.
- [x] `go build -o /tmp/cliproxyapi-test-output-0723 ./cmd/server` passed.
- [x] `bash -n deploy/scripts/run-grok-inspection.sh` passed.
- [x] No `internal/translator/**` files changed.

## Production gates

- [x] Image revision is `0672a88e4412aa2d3cc2c8697cdc963f0acc7a72` and the container is running.
- [x] Management API reports `disable-cooling: false` and `save-cooldown-status: true`.
- [x] Management API reports `routing.strategy: round-robin`.
- [x] Deployed inspection script checksum matches the tracked script.
- [x] Inspection timer is active and the low-concurrency service completed without apply failures.
- [x] 1344 remaining freshly healthy recovery targets are enabled.
- [x] 481 fresh reauth targets were backed up, checksum-verified, deleted, and verified absent.
- [x] Quota-exhausted and probe-error classes were not deleted.

## Traffic-dependent follow-up

- [ ] Confirm a real streamed free-usage event creates a `.cds` cooldown file and succeeds through another credential.
- [ ] Compare client-visible 429 frequency after representative `/v1/messages` traffic resumes.

The unchecked observations require organic or user-authorized synthetic traffic; no client key or credential was read to manufacture a production request.

## 2026-07-24 durable systemd follow-up

- [x] `git diff --check` passed.
- [x] `bash -n deploy/scripts/remote-deploy.sh deploy/scripts/run-grok-inspection.sh` passed.
- [x] Base production compose render passed.
- [x] Split-proxy production compose render passed.
- [x] Tracked service and timer passed Linux `systemd-analyze verify`.
- [x] `go build -o /tmp/cliproxyapi-test-output-0724 ./cmd/server` passed.
- [x] Service policy includes `permission_denied` and excludes `quota_exhausted` and `probe_error`.
- [x] Production manual safe apply disabled 112/112 permission-denied credentials with no apply failures.
- [x] Production unit backup exists and the five-minute timer remains active.
