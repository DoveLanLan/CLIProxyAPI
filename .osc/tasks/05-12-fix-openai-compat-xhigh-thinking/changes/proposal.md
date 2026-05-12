# Proposal: Fix OpenAI Compat XHigh Thinking Preservation

- Date: 2026-05-12
- Owner(s): hewei / Codex
- Stakeholders: CLIProxyAPI operators using OpenAI-compatible providers
- Status: Proposed

## Context / Problem

OpenAI-compatible models configured in `openai-compatibility.*.models[]` currently receive proxy-side default thinking metadata of `["low","medium","high"]` when no explicit `thinking` block is present. A namespaced DeepSeek model such as `deepseek-ai/deepseek-v3.2` is therefore treated as not supporting `xhigh`, so budget-derived `xhigh` requests are clamped to `high` before reaching the upstream provider.

## Goals (Why/What)

- Preserve `xhigh` for OpenAI-compatible configured models unless the model explicitly restricts thinking levels.
- Avoid requiring every operator to duplicate a `thinking.levels` block for namespaced compatible models.
- Keep exact explicit config authoritative when a model entry provides `thinking`.

## Constraints

- Keep the canonical thinking pipeline intact: parse to `ThinkingConfig`, validate against model metadata, then provider-specific apply.
- Do not edit `internal/translator/**`.
- Do not commit secrets or local `config.yaml` changes.
- Preserve hot-reload behavior for OpenAI-compatible model metadata.

## Non-goals

- Do not change official OpenAI/Codex provider static model behavior.
- Do not add a provider-specific DeepSeek special case in request translators.

## Proposed Approach (high-level)

Centralize OpenAI-compatible default thinking metadata construction in `sdk/cliproxy/service.go`. Prefer explicit model config, reuse static level metadata when it is directly applicable, and otherwise use a permissive OpenAI-compatible default that includes `none`, `low`, `medium`, `high`, and `xhigh` with zero allowed. Add focused regression coverage proving that a namespaced DeepSeek compatible model preserves `xhigh`.

## Risks & Mitigations

- Risk: Some OpenAI-compatible upstreams may reject `xhigh`.
  - Mitigation: This only affects callers who request `xhigh`; the proxy should not silently lower a caller's requested effort for user-configured compatible upstreams.
- Risk: Over-inheriting static budget-only metadata could break OpenAI-format compatible requests.
  - Mitigation: Only reuse static metadata when it has discrete levels suitable for OpenAI-compatible reasoning fields.

## Open Questions (max 3)

- None.
