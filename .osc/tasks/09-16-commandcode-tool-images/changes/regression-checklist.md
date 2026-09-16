# Regression evidence

- PASS: focused DeepSeek and NormalizeOpenAIToolImages tests through OSC.
- PASS: `go test ./...`.
- PASS: `go test -race ./internal/runtime/executor/... -run "DeepSeekClaude|NormalizeOpenAIToolImages" -count=1`.
- PASS: `go build -o test-output ./cmd/server && rm test-output`.
- PASS: formatting and `git diff --check`.
- Tests cover strict rejection of tool-role images, both execution modes, mixed
  text/image results, multiple tool results, trailing image groups, metadata,
  no-op requests, idempotence, and unchanged reasoning recovery.

## Claude Code CLI live test

Version: 2.1.273. Model: deepseek-v4.1-flash. Read-only tool allowlist; bare mode,
no plugins/MCP/session persistence. Credentials consumed from existing settings
in memory, never committed or printed. Synthetic image contains VISION CHECK
7392 and a red square, blue circle, green triangle from left to right; none of
these answers were supplied in the prompt or filename.

- Before: production eb923914, session b9af3397-5c0b-4561-b444-d69da51882fd.
  Read succeeded; image follow-up failed with API Error: 400 Invalid input.
- After (patched local binary -> production Chat endpoint -> Command Code):
  session e087678b-b0d2-4db3-8fb8-f2a5e441e69a. Two turns, exit 0, is_error false,
  correct text and all three colors/shapes. Local requests c3cbe90e/e638e1f0 both 200.
- Direct deployed production test: pending.

OSC shell startup emits existing fzf/zsh warnings; gate exit status is zero.
