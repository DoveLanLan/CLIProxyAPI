# Regression Checklist

- [x] Confirmed Antigravity OAuth account was loaded and registered on bytevirt.
- [x] Correlated the original Claude Code request with upstream HTTP 400.
- [x] Confirmed the exact missing `items` schema error without recording credentials.
- [x] Added focused top-level and nested array-schema tests.
- [x] Preserved explicit array `items` schemas.
- [x] `go test ./internal/util` passed.
- [x] `go test ./...` passed.
- [x] `go build -o test-output ./cmd/server && rm test-output` passed.
- [x] Built Linux amd64 binary and verified local/remote SHA-256 match.
- [x] Backed up the previous bytevirt binary before replacement.
- [x] Minimal Claude-to-Antigravity tool request returned HTTP 200 and `SMOKE_OK`.
- [x] No auth tokens, API keys, or full request bodies were persisted in repository artifacts.

## Remaining Risk

The original full Claude Code conversation may contain additional tool schemas. The sanitizer now covers all missing-items arrays, but the normal Claude Code command should still be re-run once by the operator.
