# Regression Checklist: Integrate CPA-Manager Panel and Usage Monitoring

- Date: 2026-05-14
- Related: proposal.md, spec.md, tasks.md

## Gates

- Format: `gofmt -w <changed-go-files>`
- Tests: `go test ./...`
- Build: `go build -o test-output ./cmd/server && rm test-output`

## Results

- `gofmt -w ...`: passed
- `go test ./internal/redisqueue ./internal/api/handlers/management ./sdk/cliproxy/auth ./sdk/cliproxy/usage ./internal/usage`: passed
- `go test ./...`: passed
- `go build -o test-output ./cmd/server && rm test-output`: passed

## Rollback Notes

- Revert this task's commit(s) to restore the previous management panel source and remove the usage queue endpoints.
- If already deployed, stop/remove the `cpa-manager` service and its private Tailscale port.
- No data migration is required inside CPA. CPA-Manager SQLite data lives under its own `/data` volume.
