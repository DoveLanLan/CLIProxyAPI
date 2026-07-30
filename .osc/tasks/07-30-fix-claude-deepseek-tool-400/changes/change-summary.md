# Change summary

## Diagnosis

The observed failures do not have one universal cause:

- Two older DeepSeek transcripts preserve the upstream error that thinking-mode
  tool continuations must pass back `reasoning_content`.
- The existing production workaround ensures the field exists but may populate it
  from visible text or `[reasoning unavailable]`; it does not recover the exact
  unsigned Claude thinking text.
- A text tool-result continuation returns 200, while the same minimal history with
  a PNG block consistently returns 400 from the text-only upstream.
- Plain text, single tools, sequential tools, parallel tools, empty signatures, and
  reordered tool results all pass in small controlled requests. These results do
  not support globally disabling reasoning or serializing every tool call.
- The same opencode upstream has accepted 275,697 input tokens. A later failure at
  about 169,627 reported input tokens therefore does not establish a fixed context
  limit.
- After accounting for the WSL/VPS clock offset, the latest generic local 400
  correlates with an upstream pre-response failure. Claude Code retained only
  `error: unknown`, so request-error propagation still loses diagnostic detail.

## Repository change

- Restore exact Claude thinking for DeepSeek OpenAI-compatible tool-call assistant
  messages before strict message-link normalization.
- Match through tool-call IDs, support parallel calls, preserve existing non-empty
  reasoning, and skip ambiguous duplicate IDs.
- Apply the restoration only to Claude-origin requests whose base model is
  DeepSeek. No file under `internal/translator/**` is changed.
- Reject Claude image blocks and forced named tool choice on the same DeepSeek path
  before any upstream request. These validation failures are request-scoped and do
  not cool or rotate credentials.
- Preserve upstream response headers on OpenAI-compatible status errors and include
  a sanitized upstream request ID in Claude-compatible JSON errors. Valid upstream
  JSON still supplies the client-visible error type, message, string code, and model
  identifier when present.
- Cover direct helper behavior plus non-streaming and streaming executor paths,
  capability failures, upstream request IDs, and Claude error conversion.

The repository patch is formatted, fully tested, race-checked on the affected paths,
and build-verified. Commits `aa2cbf95` and `5bf14e5d` were pushed to `main` and
deployed through the immutable image workflow. Production runs image
`sha-5bf14e5d21925dbd915336cf37ac0f3b46aeb20e`, whose OCI revision matches the
commit.

The OpenCode Go upstream still returns a generic structured message for the two
unsupported request shapes and does not provide the actual routed model, published
context limit, or request ID. The gateway does not invent those unavailable values;
the upstream provider must add them for complete diagnostics.

## Active local mitigations

- `/root/.claude/settings.deepseek-pro.json` sets
  `DISABLE_INTERLEAVED_THINKING=1` only for the DeepSeek profile.
- Claude Code 2.1.220 has an undocumented `--thinking` parser accepting
  `enabled`, `adaptive`, and `disabled`. Neither adaptive nor disabled is made
  permanent: the former changes the verified reasoning mode, while the latter
  would unnecessarily turn reasoning off.
- The same profile keeps `alwaysThinkingEnabled: true`, has no
  `CLAUDE_CODE_EFFORT_LEVEL`, and installs a DeepSeek-only `PreToolUse` guard for
  image/PDF `Read` calls.
- `/root/.claude/hooks/deepseek-text-only-read.sh` denies common image formats and
  PDF before Claude Code can create a visual block.
- `/root/.zshrc` is unchanged: `--effort max` remains before `"$@"`, so a trailing
  explicit `--effort xhigh` or `--effort high` wins.

## Active production mitigation

The bytevirt configuration now uses `streaming.bootstrap-retries: 1`. The watcher
reloaded it without a container restart. This retry is bounded to a failure before
the first downstream payload and does not address protocol-invalid requests.

The final deployment remounted the production configuration. Host and container
views now have the same inode and both report `bootstrap-retries: 1`.

## Credential containment

The exposed OpenCode upstream credential was removed from the active production
configuration and its `opencode new` provider entry was disabled. Plain text and
reasoning tool continuation both remained 200 through the remaining DeepSeek
providers after removal. The pre-change values remain only in two root-owned mode
600 timestamped backups for emergency rollback.

Provider-side revocation/regeneration is still required because disabling and
removing a credential from CLIProxyAPI cannot invalidate it at OpenCode. No logged-in
OpenCode control-plane session or supported key-rotation API is available on this
machine. Installing a replacement therefore requires an operator-authenticated
OpenCode console session; neither the old nor a future replacement value belongs in
chat, logs, or repository files.
