# PRD: Built-in Command Code provider

## Problem

The user's memory-constrained VPS cannot accommodate a separate Command Code
Node.js proxy. Integrate the upstream protocol into the existing CPA process.

## Goals

- Confirmed: a built-in Go provider, not a sidecar or dynamic plugin.
- Reuse CPA request/response translation, credential scheduling, and usage reporting.
- Process streaming responses incrementally; bound retained per-key session state.
- Preserve existing providers and the canonical thinking pipeline.

## Repository Evidence

- `internal/runtime/executor/kimi_executor.go`: provider execution and shared translation pattern.
- `internal/watcher/synthesizer/config.go`: configured API-key credential synthesis.
- `sdk/cliproxy/service.go`: executor and model registration.
- `internal/config/config.go`: provider configuration conventions.
- `AGENTS.md`: supporting code belongs under executor/helps; no new post-connect timeouts.
- Reference implementation: MAXeaglet/commandcode-proxy, inspected HEAD
  `6b845b6f169170b6164b71bfacdd9cb2b762ce39`; live authenticated compatibility remains unverified.

## First Release (Confirmed)

- YAML-configured Command Code API keys, multiple credentials, model aliases and hot reload.
- Chat Completions, Messages, and Responses through existing CPA protocol surfaces.
- Streaming and non-streaming text, tools, image input where supported, thinking, and usage.
- Upstream initialization/session handling and cached model discovery with local-model policy respected.
- No dedicated management UI in the first release.

## Non-goals

- Extra services, Node.js runtime, dynamic plugin deployment, or VPS deployment without authorization.
- Fabricated Anthropic signatures or automatic replay after meaningful downstream output.
- Guaranteed memory savings without measurement; full Responses stateful API parity.

## Acceptance criteria

- Existing CPA binary reaches the Command Code upstream without a sidecar.
- Tests cover wire conversion, SSE fragmentation, tool continuations, cancellation,
  usage, bounded retained state, credential isolation and config reload.
- Focused tests and required server build pass; broader regressions are recorded.
- Live account tests are explicitly distinguished from mock-based checks.

## Confirmed Decision

The user approved YAML-only configuration without a dedicated management page.
Implementation is authorized. No deployment or real credential changes are authorized.
