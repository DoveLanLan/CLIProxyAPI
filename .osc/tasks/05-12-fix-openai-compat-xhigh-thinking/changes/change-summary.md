# Change Summary: Fix OpenAI Compat XHigh Thinking Preservation

- Date: 2026-05-12
- Owner(s): hewei / Codex
- Related: spec.md, tasks.md

## What changed

- OpenAI-compatible configured models now resolve thinking metadata through a single helper in `sdk/cliproxy/service.go`.
- Nil `thinking` metadata now defaults to `none`, `low`, `medium`, `high`, and `xhigh` with zero allowed instead of only `low`, `medium`, and `high`.
- Static discrete-level thinking metadata is reused when a configured name or alias matches a known static model.
- Added regression tests for namespaced DeepSeek and prefixed production-style registration.
- Updated `config.example.yaml` to document the new default.

## Why

The proxy was over-constraining user-configured OpenAI-compatible upstream models. That caused budget-derived `xhigh` requests for models such as `deepseek-ai/deepseek-v3.2` to be clamped to `high` before reaching the provider.

## Notable decisions

- Explicit config remains authoritative; a model entry with its own `thinking` block is not widened.
- The fallback is generic for OpenAI-compatible providers rather than a one-off DeepSeek special case, so future namespaced compatible models do not regress the same way.
