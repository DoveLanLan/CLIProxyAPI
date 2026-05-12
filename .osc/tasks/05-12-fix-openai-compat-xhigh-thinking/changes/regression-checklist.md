# Regression Checklist: Fix OpenAI Compat XHigh Thinking Preservation

- Date: 2026-05-12
- Related: spec.md, tasks.md

## Gates (from Repo Snapshot)

- Build: `go build -o test-output ./cmd/server && rm test-output`
- Tests: focused package tests plus top-level thinking regression tests
- Format: `gofmt -w sdk/cliproxy/service.go sdk/cliproxy/service_openai_compat_thinking_test.go`

## Results

- PASS: `go test ./sdk/cliproxy -run 'TestOpenAICompat.*Thinking|TestOpenAICompat.*DeepSeek|TestOpenAICompatRegisterModelsForAuthPrefixedDeepSeekPreservesXHigh'`
- PASS: `go test ./internal/watcher/diff -run TestComputeOpenAICompatModelsHash_ThinkingChangesHash`
- PASS: `go test ./sdk/cliproxy ./internal/thinking ./internal/watcher/diff`
- PASS: `go test ./test -run TestThinking`
- PASS: `go test ./...`
- PASS: `go build -o test-output ./cmd/server && rm test-output`

## Manual Checks

- Confirmed no edits were made under `internal/translator/**`.
- Confirmed `config.yaml` was not edited.
- Confirmed prefixed model ID `nih/deepseek-ai/deepseek-v3.2(64000)` is covered by a regression test.
