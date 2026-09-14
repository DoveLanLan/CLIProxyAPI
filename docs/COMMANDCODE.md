# Native Command Code Provider

CPA can connect directly to Command Code using its built-in Go executor. No Node.js
runtime, sidecar container, additional listener, or dynamic plugin is required.

## Configuration

Add a separate provider entry to your existing YAML configuration:

```yaml
commandcode-api-key:
  - api-key: "user_REPLACE_ME"
    prefix: "cc"
    models:
      - name: "YOUR_UPSTREAM_MODEL_ID"
        alias: "my-model"
```

Keep your usual CPA frontend API keys unchanged. Clients authenticate with the CPA
key and request `cc/my-model`; the upstream `user_...` credential remains server-side.
Use actual model IDs available to your Command Code account, not the placeholder.
Do not commit real credentials. Removing an entry disables it through the existing
config watcher; model aliases, proxy settings and priorities also support reload.

Optional fields are `base-url`, `proxy-url`, `priority`, `headers`,
`excluded-models` and `disable-cooling`. `base-url` defaults to
`https://api.commandcode.ai` and must be a root URL, not `/v1`. Multiple entries use
CPA's existing credential scheduler and cooldown handling. Configuration uses the
existing Gemini-style key/model schema; this does not route requests through Gemini.

For upstream ZDR routing, set `headers: {x-cmd-zdr: "1"}`. This is an upstream
request flag, not a guarantee about retention. It is omitted from catalog discovery.

Explicit `models` lists are recommended: they avoid upstream discovery during
startup/reload. When omitted, the provider fetches `/provider/v1/models` during
credential registration, with a five-minute per-credential cache. There is no new
background polling loop. Discovery failure registers no guessed models. `--local-model`
disables Command Code discovery, so explicit models are required in that mode.

## Supported Behavior

- Existing Chat Completions, Claude Messages and Responses HTTP interfaces.
- Streaming and non-streaming text, reasoning, function tools and tool results.
- Image input using models that support it (not image generation).
- CPA thinking normalization, token usage including cache reads/writes, model
  aliases, request cancellation and existing credential scheduling.
- Native upstream NDJSON and SSE-framed compatibility input, normalized to CPA's
  existing response translators. Non-streaming generation aggregates that stream.
- Provider-local recovery of exact Claude thinking attached to tool-call history,
  without changing shared signature validation or fabricating Anthropic signatures.

The provider does not implement OAuth login, a dedicated management page, upstream
WebSocket transport, Responses storage/retrieval, background Responses, compact,
built-in server-side tools, or media generation. Send full conversation history for
Responses continuations; `previous_response_id` and background execution are rejected.
Token counting uses CPA's local estimate, not an exact upstream tokenizer.
Tool results support text; image input support does not imply image tool results.
Custom tool names are preserved consistently, not rewritten to CLI-owned aliases.

## Resource and Failure Boundaries

- Requests: at most 8 MiB before and after shared request conversion.
- Upstream event: at most 4 MiB; at most 1,024 distinct tool calls per response.
- Non-stream aggregation: at most 16 MiB of normalized chunks.
- Claude/Responses translated streams: at most 16 MiB of normalized chunks because
  existing shared translators may retain text/tool output. OpenAI streaming does
  not retain the full answer in the new provider.
- Session cache: at most 256 entries, eight-hour expiration, lazy eviction.
- Catalog cache: at most 256 entries, at most 1 MiB response / 2,048 model IDs each.
- Concurrent initialization for the same credential/endpoint is coalesced. Cached
  identities are hashed; anonymous fingerprints are deterministic across restarts.
- No new post-connect network timeout. Caller cancellation aborts upstream work.
- Truncated streams and malformed events fail explicitly. Generation is not
  replayed inside the executor after output. Existing CPA bootstrap retry rules apply.
- HTTP 402 maps to 429 for credential cooldown. Error messages avoid raw upstream
  bodies and key fragments. Upstream token usage is not zeroed when output is zero.

These are payload/state bounds, **not** a process RSS guarantee. Go allocations,
concurrent requests, shared translators and optional request logging consume memory
too. Prefer streaming, explicit model lists and conservative client concurrency on
small VPS instances; disable full request logging if it is not needed.

## Protocol Provenance and Validation

Wire behavior references `MAXeaglet/commandcode-proxy` revision
`6b845b6f169170b6164b71bfacdd9cb2b762ce39`, protocol version `1.53.1`.
The version is pinned rather than silently upgraded. Initialization uses a stable
anonymous device profile, not real host filesystem or machine identifiers. This
does not guarantee account acceptance or future upstream compatibility.

The reference project's MIT notice is retained in
[licenses/commandcode-proxy.txt](licenses/commandcode-proxy.txt).

Mock tests cover protocol conversion, tool continuations, stream fragmentation,
usage, cancellation, errors, cache isolation and registration. Before production
enablement, validate with your own account: text, multi-turn tools, thinking,
images on a supported model, streaming/non-streaming and real usage. No real-account
availability or VPS RSS reduction is established by mock tests.

## Rollback

Remove `commandcode-api-key` from YAML and let the watcher reload. No data migration
or sidecar cleanup is required. To roll back the binary, remove this configuration
first and deploy the previous CPA image using your usual deployment procedure.
