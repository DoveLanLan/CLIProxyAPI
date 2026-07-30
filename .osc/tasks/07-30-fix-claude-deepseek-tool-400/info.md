# Tech notes

## Architecture decisions

- Keep the canonical translator security rule that rejects unknown or unsigned
  cross-provider signatures.
- Restore unsigned thinking only in the model-specific DeepSeek executor path, where
  the target protocol requires `reasoning_content` on tool continuation.
- Keep support code under `internal/runtime/executor/helps/`.
- Treat image rejection and transient upstream failures as separate problems.

## Risks and mitigations

- Replaying reasoning across a different credential can be unsafe. Session affinity
  remains enabled, and restoration is limited to DeepSeek tool-call messages in the
  current request rather than a global translator relaxation.
- A retry can duplicate non-idempotent work. Streaming bootstrap retry is limited to
  one attempt and only applies before response payload bytes are sent.
- Blocking image `Read` reduces functionality only for the DeepSeek settings profile;
  other Claude profiles remain unchanged.
- `DISABLE_INTERLEAVED_THINKING` removes the beta negotiation but does not disable
  model reasoning or alter effort precedence.

## Rollback plan

- Restore timestamped backups of the local DeepSeek settings, hook, and `.zshrc`.
- Restore the timestamped VPS `config.yaml` backup; the watcher reloads it.
- Revert the executor/helper changes if the gateway patch causes a regression.

## Sensitive-data handling

- Do not persist client tokens, upstream API keys, Authorization headers, request
  bodies, or project tool output in task artifacts.
- The local debug file is mode 600 and must never be pasted verbatim.
