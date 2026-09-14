# Specification

## Configuration and Routing

- commandcode-api-key accepts api-key, base-url, prefix, proxy-url, priority,
  headers, models (name/alias/display-name/force-mapping), excluded-models and
  disable-cooling. Empty base-url selects https://api.commandcode.ai.
- Configured credentials participate in existing scheduling and hot reload.
- Configured model lists take precedence. Discovery is scoped by credential and
  endpoint, cached, and disabled by --local-model. Failed discovery must not invent
  model availability. Static lists are recommended for predictable startup.
- Existing protocols use CPA translators; no new frontend routes or UI.

## Execution

- Key format is user_ followed by alphanumeric, underscore or hyphen characters.
- Initialize fingerprint/lifecycle once per cached credential interval. No host
  filesystem information is sent. Do not silently rotate claimed protocol version.
- Convert system/developer blocks, text/images, reasoning, tools/results, choices,
  and cache hints to the upstream envelope; generation is always streaming.
- Decode native NDJSON (the inspected upstream format) and optional SSE framing.
  Translate text, reasoning, tool events, finish reason and usage. Reject malformed
  events, truncated streams and unsupported operations explicitly.
- Never fabricate thinking signatures or wipe upstream usage because output is zero.
- Cancellation closes upstream requests; channel sends respect context cancellation.
- Bound SSE event size, retained session entries, discovery response, and non-stream
  aggregation. Do not buffer a full answer for a streaming client.

### Concrete Resource Boundaries

Requests: 8 MiB before/after translation; events: 4 MiB; non-stream aggregation and
input to stateful shared response translators: 16 MiB; sessions/catalogs: 256 entries
each; tools: 1,024 with IDs/names no longer than 256 bytes. Retained event strings
must be cloned so small identifiers cannot pin large event buffers. Raw OpenAI
streaming does not accumulate answer text. These are not process RSS guarantees.

### Discovery Policy

Catalog lookup happens during credential registration, with a five-minute cache.
No background polling loop is introduced. Explicit YAML models avoid network
discovery and are required with --local-model. Failed discovery invents no models.

## Acceptance / Validation

Focused tests cover config synthesis, aliases/registration, session isolation,
initialization coalescing, SSE fragmentation, tools, usage, error handling,
cancellation, response translation and limits. Run gofmt, required server build,
focused/race tests and broader tests where practical. Record blocked live validation.
