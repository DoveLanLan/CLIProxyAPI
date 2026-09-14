# Built-in Command Code Provider

## Context / Goals

Replace a separate Node proxy with native Go execution in CPA. Reuse existing
protocol translation, auth scheduling, hot reload, and usage reporting. YAML-only
configuration is confirmed. No deployment changes or new runtime dependencies.

## Approach

Add commandcode-api-key entries using the existing Gemini-style key/model schema.
Register a native executor and per-auth models. Normalize incoming protocols to
OpenAI chat, build the Command Code envelope, and normalize upstream SSE back to
OpenAI chunks before existing response translation. Keep wire and session helpers
in executor/helps. Use configured models preferentially and bounded model discovery
when no models are configured; local-model must disable discovery.

## Constraints / Risks

Preserve canonical thinking, no fabricated signatures, no post-connect timeouts,
no retry after downstream content. Session state must be bounded, keyed by exact
credentials and endpoint, and initialization coalesced. Never log key fragments.
Reference protocol may drift; pin the inspected revision and retain MIT notice.
Mock tests cannot establish live account access or exact VPS memory savings.

## Non-goals

Management UI, OAuth acquisition, sidecars, WebSocket upstream, stateful Responses
storage/compact, and upstream billing correction are outside scope.
