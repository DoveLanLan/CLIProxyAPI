# Change Summary

Implemented an in-process Command Code provider configured via `commandcode-api-key`.
No runtime dependency, sidecar, new listener or management UI was introduced.

## Integration

- Added YAML parsing/sanitization, watcher synthesis and redacted change reporting.
- Registered native executor and per-auth models; integrated exact-key alias lookup.
- Reused shared protocol translation, canonical thinking, scheduling and usage.
- Added native NDJSON / SSE decoding, CLI envelope conversion, bounded session and
  catalog caches, cancellation, tool deltas and finish/error handling.
- Added local-model discovery policy, configured-model precedence and catalog cache.
- Added usage/cache accounting, valid Retry-After propagation, and explicit rejection
  of unsupported stateful/background Responses and built-in server-side tools.
- Kept provider-specific support files under executor/helps. Shared translator files
  and dependency manifests were not changed.
- Added configuration documentation and the reference project's MIT attribution.

## Important Decisions

Upstream generation is native NDJSON, not standard SSE; both framings are tested.
Claude streaming requires an explicit stream flag in the original translator input,
including direct SDK calls. Tool identifiers/names are copied so small strings do not
retain large event buffers. Existing non-OpenAI translators may accumulate output,
so their normalized input is capped rather than claiming constant-memory behavior.
Thinking signatures are never fabricated. No post-connect network timeout was added.

## Verification

All repository tests passed. Eighteen top-level Command Code tests (plus subtests)
cover wire formats, three client protocols, tools, usage, cancellation, errors,
credential isolation, model aliases, registration, rotation and resource limits.
Focused race tests, native server build, and Linux amd64 CGO-disabled build passed.

Benchmark on Apple M5 / darwin arm64: text-event conversion measured 9,787 ns/op,
2,321 B/op, 38 allocations/op in one run. This measures allocations per event, not
retained memory or VPS RSS; concurrent build activity also affects timing.

## Remaining Release Checks

No real Command Code credential was used, no VPS was modified, and no commit or
deployment was performed. Live protocol acceptance, supported models, quota behavior
and actual VPS memory consumption remain deployment-time checks.
