# Regression

- PASS: `go test ./internal/runtime/executor/... -run DeepSeek -count=1` via OSC.
- PASS: `gofmt -w .`; no unrelated Go files changed.
- PASS: `git diff --check`.
- PASS: `go test -race ./internal/runtime/executor/... -run 'DeepSeekClaude' -count=1`.
- PASS: `go test ./...` via OSC.
- PASS: `go build -o test-output ./cmd/server && rm test-output` via OSC.
- Covered: translated PNG data URL, nested tool result, streaming, model prefix,
  upstream rejection, existing named-tool validation and reasoning tests.
- Not run: live production smoke test; no deployment requested or performed.

The OSC login shell prints pre-existing fzf/zsh initialization warnings; Go tests
and build still complete with exit code zero.
