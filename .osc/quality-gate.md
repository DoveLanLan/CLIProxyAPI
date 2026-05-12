# Quality Gate: Fix OpenAI Compat XHigh Thinking Preservation

- Date: 2026-05-12

## Assumptions

- The change scope is backend/model-registration logic for OpenAI-compatible configured providers.
- `config.yaml` is local and secret-bearing; it is intentionally not modified.

## Suspected Change Scope

- `sdk/cliproxy/service.go`: OpenAI-compatible model registration and thinking metadata defaults.
- `sdk/cliproxy/service_openai_compat_thinking_test.go`: regression coverage.
- `config.example.yaml`: documented default.

## Detected Gates

- Gate Name: PR server build
  - Confidence: High
  - Evidence: `.github/workflows/pr-test-build.yml` runs `go build -o test-output ./cmd/server` then removes `test-output`.
- Gate Name: Restricted translator path guard
  - Confidence: High
  - Evidence: `.github/workflows/pr-path-guard.yml` fails PRs touching `internal/translator/**`.
- Gate Name: Go formatting
  - Confidence: High
  - Evidence: `AGENTS.md` requires `gofmt -w .` after Go changes.
- Gate Name: Focused Go tests
  - Confidence: Medium
  - Evidence: package test layout and `AGENTS.md` command examples.

## Suggested Gate Run (Local)

1. `gofmt -w sdk/cliproxy/service.go sdk/cliproxy/service_openai_compat_thinking_test.go`
2. `go test ./sdk/cliproxy -run 'TestOpenAICompat.*Thinking|TestOpenAICompat.*DeepSeek|TestOpenAICompatRegisterModelsForAuthPrefixedDeepSeekPreservesXHigh'`
3. `go test ./internal/watcher/diff -run TestComputeOpenAICompatModelsHash_ThinkingChangesHash`
4. `go test ./sdk/cliproxy ./internal/thinking ./internal/watcher/diff`
5. `go test ./test -run TestThinking`
6. `go test ./...`
7. `go build -o test-output ./cmd/server && rm test-output`

## Results

- PASS: `gofmt -w sdk/cliproxy/service.go sdk/cliproxy/service_openai_compat_thinking_test.go`
- PASS: `go test ./sdk/cliproxy -run 'TestOpenAICompat.*Thinking|TestOpenAICompat.*DeepSeek|TestOpenAICompatRegisterModelsForAuthPrefixedDeepSeekPreservesXHigh'`
- PASS: `go test ./internal/watcher/diff -run TestComputeOpenAICompatModelsHash_ThinkingChangesHash`
- PASS: `go test ./sdk/cliproxy ./internal/thinking ./internal/watcher/diff`
- PASS: `go test ./test -run TestThinking`
- PASS: `go test ./...`
- PASS: `go build -o test-output ./cmd/server && rm test-output`

## Final Self-Review

- Security & secrets: no secret-bearing `config.yaml` edits.
- Edge cases & error handling: explicit per-model thinking remains authoritative; prefixed models are covered.
- Backward compatibility / migrations: no data or schema migrations.
- API/contract compatibility: only prevents proxy-side over-clamping for compatible upstreams.
- Observability: existing thinking debug logs still apply.
- Config/env changes: no new fields; example comment updated.
- Performance risk: negligible; metadata resolution runs during model registration.
- Rollback plan: revert touched code/docs or explicitly restrict a model's `thinking.levels`.

## PR-ready checklist

- [x] `gofmt -w sdk/cliproxy/service.go sdk/cliproxy/service_openai_compat_thinking_test.go`
- [x] `go test ./sdk/cliproxy -run 'TestOpenAICompat.*Thinking|TestOpenAICompat.*DeepSeek|TestOpenAICompatRegisterModelsForAuthPrefixedDeepSeekPreservesXHigh'`
- [x] `go test ./internal/watcher/diff -run TestComputeOpenAICompatModelsHash_ThinkingChangesHash`
- [x] `go test ./sdk/cliproxy ./internal/thinking ./internal/watcher/diff`
- [x] `go test ./test -run TestThinking`
- [x] `go test ./...`
- [x] `go build -o test-output ./cmd/server && rm test-output`
- [x] Confirm no `internal/translator/**` changes.
