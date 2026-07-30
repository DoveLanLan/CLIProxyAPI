# Specification

## DeepSeek reasoning replay

Given a Claude request containing an assistant message with one or more `tool_use`
blocks and one or more non-empty `thinking` blocks, the DeepSeek OpenAI-compatible
execution path must place the exact joined thinking text in the translated assistant
message's `reasoning_content`, even when the Claude thinking signature is absent or
empty.

The restoration must:

- apply only when the requested base model begins with `deepseek-`;
- match source and translated assistant messages through tool call IDs;
- support multiple tool calls in one assistant message;
- leave existing non-empty compatible `reasoning_content` unchanged;
- fall back to the existing normalizer when no unambiguous source reasoning exists;
- avoid changing the shared translator signature policy.

## Local DeepSeek profile

- Keep `alwaysThinkingEnabled: true`.
- Keep the wrapper's default `--effort max` and trailing user arguments.
- Do not define `CLAUDE_CODE_EFFORT_LEVEL`.
- Set `DISABLE_INTERLEAVED_THINKING=1` only in the DeepSeek settings profile.
- Deny `Read` when `file_path` ends in a known raster/vector image extension,
  case-insensitively, with a clear model-visible reason.

## Streaming recovery

Production may retry once when an upstream streaming attempt fails before any payload
bytes are forwarded. No retry is allowed after payload emission.

## Observability

Tests and reports may record status, model, message roles, block types, token counts,
lengths, and request IDs. They must not record credentials or full sensitive payloads.

When a safe structured upstream error body is available, the Claude-compatible
client response must retain the useful message and request identifier. It must not
include credentials, authorization headers, or full request payloads. If the
upstream omits actual model or context-limit metadata, the gateway must not invent
those values.

## DeepSeek capability validation

For Claude-origin requests translated to a DeepSeek OpenAI-compatible model:

- image content must fail before upstream dispatch with a structured text-only-model
  compatibility error;
- a forced named tool choice unsupported by the configured upstream must fail before
  dispatch with a structured unsupported-parameter error;
- automatic, required, and absent tool-choice modes must retain existing behavior;
- validation must apply to streaming and non-streaming execution without changing
  other providers.

## Delivery and credential hygiene

- Rotate only the exposed upstream credential; do not print either old or new value.
- Confirm the replacement through a health request and hot reload or controlled
  restart before revoking or removing the old credential when the provider supports
  overlap.
- Commit and push the source/docs change to `main`, wait for the immutable image
  workflow and production deployment, and verify the running OCI revision.
