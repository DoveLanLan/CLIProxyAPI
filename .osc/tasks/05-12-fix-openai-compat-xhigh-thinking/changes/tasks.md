# Tasks: Fix OpenAI Compat XHigh Thinking Preservation

- Date: 2026-05-12
- Owner(s): hewei / Codex
- Related: spec.md, proposal.md

## Assumptions

- The affected route is OpenAI-compatible configured provider registration in `sdk/cliproxy/service.go`.
- OpenAI-compatible third-party providers should receive caller-requested effort values unless explicit config says otherwise.

## Checklist

- [x] 1) Centralize OpenAI-compatible thinking metadata defaulting
  - Target: `sdk/cliproxy/service.go`
  - Change: prefer explicit config, safe static discrete-level metadata, then permissive OpenAI-compatible default with `xhigh`.
  - Verify: focused unit tests.

- [x] 2) Add regression coverage
  - Target: `sdk/cliproxy/*_test.go`
  - Change: assert namespaced DeepSeek default supports `xhigh` and `ApplyThinking` preserves `xhigh` from a large suffix budget.
  - Verify: `go test ./sdk/cliproxy -run TestOpenAICompat`

- [x] 3) Update example docs
  - Target: `config.example.yaml`
  - Change: document the new default levels.
  - Verify: review diff.

- [x] 4) Run quality gates
  - Target: repository
  - Change: gofmt, focused tests, compile server.
  - Verify: `gofmt`, `go test`, `go build -o test-output ./cmd/server && rm test-output`.

## Notes

- Do not edit `config.yaml`; it contains local deployment secrets and is ignored.
- Implemented with `resolveOpenAICompatThinking`, which preserves explicit config, reuses static discrete-level metadata when available, and otherwise defaults compatible models to `none/low/medium/high/xhigh`.
