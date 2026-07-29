# Tasks: prevent Cloudflare 524 during xAI Resin bootstrap

- [x] Add a joined bootstrap SSE keep-alive helper using the configured interval.
- [x] Start it around Claude-compatible synchronous stream setup.
- [x] Preserve HTTP startup errors before the first heartbeat and emit SSE
      terminal errors after the response is committed.
- [x] Add focused helper and Claude handler regression tests.
- [x] Add a sourceable Grok timer deployment helper and shell regression tests.
- [x] Add `ENABLE_GROK_INSPECTION_TIMER` to the environment template and docs.
- [x] Set the production VPS switch to `false` without exposing other values.
- [x] Run `gofmt -w .`.
- [x] Run focused Go tests, Resin retry tests, shell tests, and shell syntax checks.
- [x] Run `go test ./...`.
- [x] Run `go build -o test-output ./cmd/server && rm test-output`.
- [x] Confirm `internal/translator/**` is untouched and `git diff --check` passes.
- [x] Write regression, rollback, summary, and quality-gate artifacts.
- [x] Commit and push, then verify the GitHub Actions deployment and production.
