# Quality gate: Claude Code DeepSeek tool continuation 400s

Date: 2026-07-30
Task: `.osc/tasks/07-30-fix-claude-deepseek-tool-400`
Status: PASS LOCALLY; SOURCE DEPLOYMENT PENDING

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

The source patch is verified but uncommitted and not deployed. Production remains on
`ghcr.io/dovelanlan/cliproxyapi:sha-d6027f...`; the config and local-profile
mitigations are active independently.

## Security note

No tokens or full sensitive request bodies are recorded in repository artifacts.
The local debug log remains mode 600 and must not be copied verbatim. An upstream
credential appeared in transient terminal output during diagnosis; rotate that
credential even though it was not persisted in these changes.
