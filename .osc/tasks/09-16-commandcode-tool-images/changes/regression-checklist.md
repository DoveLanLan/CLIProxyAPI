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
- PASS: direct deployed production test, 2026-09-16 18:14 Asia/Shanghai.
  Image: `ghcr.io/dovelanlan/cliproxyapi:sha-bff1543200ec7415e408145860772df82a834f8a`.
  Claude session `5ddaae23-e15c-46ec-8d78-bd8ed38c3cf5`, two turns, exit 0,
  `is_error: false`, `passed: true`. Correctly read VISION CHECK 7392, red square,
  blue circle, green triangle. Prompt did not disclose these answers.
  Server requests `8908bd22` and `96d84588` returned 200 and selected commandcode.
- PASS: direct production non-streaming Chat request with mixed text/image result
  and a second parallel tool result: request `872de0c7`, HTTP 200, correct answer.
- PASS: GitHub docker-image run 35083627069 and deploy-production run 35083768330.
- PASS: `node --check` on the reusable CLI smoke script.
- Local test relay and binary stopped after acceptance; production config unchanged.

## Repeatable CLI acceptance

Use `rsvg-convert vision-card.svg -o /tmp/vision-card.png` from this directory,
then run `node claude-vision-smoke.cjs <existing-settings.json> /tmp/vision-card.png`.
The settings file must contain the existing CPA base URL and auth environment.
The script uses Claude Code's bare mode and Read-only tools, checks actual image
tool output and the answer, and exits nonzero on failure. No credentials are
written to the repository or printed. Each run makes small billable model calls.

OSC shell startup emits existing fzf/zsh warnings; gate exit status is zero.
