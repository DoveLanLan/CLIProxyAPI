# Bugfix: Fix Claude Code DeepSeek tool continuation 400s

## Problem

Claude Code 2.1.220 occasionally terminates long DeepSeek or Grok agent turns with
`API Error: 400 Bad Request` immediately after tool results. The local endpoint is
CLIProxyAPI, which translates Anthropic messages to an OpenAI-compatible upstream.

## Reproduction evidence

1. A DeepSeek transcript returned thinking plus two tool calls, accepted both tool
   results, and failed on the next continuation at 169,627 reported input tokens.
2. Older DeepSeek turns explicitly reported that `reasoning_content` must be replayed.
3. Generic 400s also occurred with Grok and non-empty signatures, so an empty
   signature is not the only cause.
4. A minimal text tool result succeeds, while the same history with a PNG image in
   the tool result consistently returns 400 from the text-only upstream.
5. Single, sequential, and parallel DeepSeek tool continuations all succeed in small
   controlled tests, including empty-signature replay.
6. The same opencode upstream previously accepted at least 275,697 input tokens, so
   the latest 169,627-token failure is not evidence of a fixed context limit.
7. VPS logs show both upstream 400s and transient pre-response/stream errors. Some
   downstream generic 400s do not preserve enough upstream detail for diagnosis.

## Expected behavior

- DeepSeek tool continuations replay the original model reasoning text.
- Transient failures before the first streamed response byte are retried safely.
- Text-only DeepSeek sessions do not send image blocks upstream.
- DeepSeek keeps max effort and reasoning enabled while avoiding unsupported
  interleaved-thinking protocol negotiation.
- Explicit `--effort xhigh` or `--effort high` remains able to override the wrapper
  default.

## Root cause assessment

There are multiple causes rather than one universal 400:

- The Anthropic-to-OpenAI translator intentionally drops unsigned thinking blocks.
  DeepSeek responses currently return thinking with no signature, so the existing
  executor normalizer inserts visible text or `[reasoning unavailable]` instead of
  replaying the original reasoning text.
- Text-only OpenAI-compatible models reject image content returned by Claude Code's
  `Read` tool.
- The opencode upstream sometimes returns generic 400 or transient 5xx/non-SSE
  failures. The current production streaming bootstrap retry is disabled.
- Claude Code's persisted `effortLevel` schema accepts through `xhigh`, not `max`;
  the existing wrapper-level `--effort max` is therefore required and must remain.

## Fix

1. Restore original Claude thinking text into DeepSeek assistant tool-call messages
   in the OpenAI-compatible executor before the strict linkage normalizer runs.
2. Add unit coverage for empty-signature Claude thinking with single and parallel
   DeepSeek tool calls.
3. Add a DeepSeek-only local PreToolUse hook that denies `Read` for image extensions.
4. Add `DISABLE_INTERLEAVED_THINKING=1` only to the DeepSeek settings file; do not
   disable reasoning or set `CLAUDE_CODE_EFFORT_LEVEL`.
5. Back up the VPS config and enable one safe streaming bootstrap retry.

## Regression tests

- [x] Existing and new OpenAI-compatible executor unit tests pass.
- [x] `gofmt -w .` completes.
- [x] `go test ./...` passes.
- [x] Required compile verification passes.
- [x] Plain text, single tool, sequential tools, and parallel tools succeed.
- [x] Image `Read` is denied locally before an image block reaches the API.
- [x] Default wrapper reports max effort.
- [x] Explicit xhigh override still works; trailing wrapper arguments preserve the same precedence for high.
