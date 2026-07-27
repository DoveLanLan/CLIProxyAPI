# Regression Checklist: Limit xAI Credential Refresh Storm

- [x] Disabled auth scheduler test passes.
- [x] Disabled auth refresh-evaluation test passes.
- [x] xAI `invalid_grant` status classification test passes.
- [x] Full `sdk/cliproxy/auth` package tests pass.
- [x] Full `internal/runtime/executor` package tests pass.
- [x] Full `go test ./... -count=1` passes.
- [x] Required server build passes.
- [x] Docker image builds for Linux AMD64 and loads on production.
- [x] Production config retains 0600 root ownership.
- [x] Management endpoint returns HTTP 200 after replacement.
- [x] Real Grok stream completes through Cloudflare.
- [x] Disabled credentials are absent from production refresh attempts.
- [x] Permanent xAI refresh failure does not retry after five minutes.
- [x] No protected `internal/translator/**` files changed.
