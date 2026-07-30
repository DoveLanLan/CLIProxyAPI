# Quality gate: Claude Code DeepSeek tool continuation 400s

Date: 2026-07-30
Task: `.osc/tasks/07-30-fix-claude-deepseek-tool-400`
Status: PASS; PRODUCTION DEPLOYMENT VERIFIED; PROVIDER REVOCATION PENDING

## Changed scope

- DeepSeek-only recovery of exact Claude thinking for OpenAI-compatible tool
  continuations.
- Pre-dispatch structured rejection for DeepSeek text-only image input and forced
  named tool choice.
- Safe Claude-compatible upstream error type/message/request-ID preservation.
- DeepSeek-only local interleaved-thinking negotiation disablement and image/PDF
  `Read` guard.
- One production pre-payload streaming bootstrap retry.
- No Claude Code binary changes, no global reasoning disablement, and no changes
  under `internal/translator/**`.

## Gates

| Command or check | Result |
|---|---|
| `gofmt -w .` | PASS |
| `go test ./internal/runtime/executor/helps ./internal/runtime/executor` | PASS |
| `go test ./sdk/api/handlers/claude` | PASS |
| Focused changed-path `go test -race` runs | PASS |
| `go test ./...` | PASS |
| `go build -o test-output ./cmd/server && rm test-output` | PASS |
| `git diff --check` | PASS |
| Changed-path guard for `internal/translator/**` | PASS |
| Credential-pattern scan of repository changes | PASS |

## Behavior verification

- Small plain-text, single-tool, sequential-tool, and parallel-tool requests pass.
- Empty signatures and interleaved tool calls alone do not reproduce generic 400.
- A PNG tool result consistently reproduces the text-only upstream 400.
- The local DeepSeek hook blocks a real image `Read` before API submission.
- Executor tests confirm image and named-tool-choice errors make no upstream request,
  remain request-scoped, and carry a structured compatibility code/model.
- Claude handler tests confirm valid upstream JSON and safe request IDs survive error
  conversion without enabling response-header passthrough.
- Default wrapper launch remains max effort; a trailing explicit xhigh override wins.
- Production watcher loaded `streaming.bootstrap-retries: 1` without restart.

## Delivery note

Commits `aa2cbf95` and `5bf14e5d` are pushed to `main`. Final Docker workflow
`30521223617` and deployment workflow `30521315577` succeeded. Production runs
`ghcr.io/dovelanlan/cliproxyapi:sha-5bf14e5d21925dbd915336cf37ac0f3b46aeb20e`;
the OCI revision matches, both public and Tailscale health checks return 200, and
host/container configuration views share one inode with `bootstrap-retries: 1`.

The final production matrix returns 200 for plain text and empty-signature reasoning
tool continuation. Image and named tool choice return local structured 400 errors
with `model_text_only` or `unsupported_tool_choice` plus the actual model identifier.

## Security note

No tokens or full sensitive request bodies are recorded in repository artifacts.
The local debug log remains mode 600 and must not be copied verbatim. The exposed
OpenCode credential is removed from active production config and its entry is
disabled; fallback production checks pass. Root-only mode 600 emergency backups
still contain it. Provider-side revocation/regeneration remains blocked on an
authenticated OpenCode control-plane session.
