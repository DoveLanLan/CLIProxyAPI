# Regression checklist

## Controlled API matrix

| Case | Result | Interpretation |
|---|---|---|
| Plain text | 200 | Baseline route is healthy. |
| One tool call and continuation | 200 | Tool continuation is not universally broken. |
| Two sequential single-tool turns | 200 | Repeated continuation is valid in a small history. |
| Two parallel calls, one tool-result message | 200 | Normal Claude parallel ordering is accepted. |
| Parallel results in separate user messages | 200 | The gateway tolerated split results in the controlled case. |
| Parallel results reversed | 200 | Ordering alone did not reproduce the failure. |
| Empty/missing thinking signature | 200 | Empty signature alone is insufficient to cause generic 400. |
| Thinking removed or synthetic signature | 200 | Signature shape alone is insufficient in the controlled case. |
| Text tool result | 200 | Text-only continuation is accepted. |
| PNG tool result | 400, repeatable | Text-only upstream rejects visual content. |
| Forced named `tool_choice` | 400, repeatable | Upstream limitation found, but not observed in normal Claude Code traffic. |
| Historical 275,697-token request | 200 | No evidence for a fixed limit below the later 169K failure. |

Requests used placeholders or locally loaded credentials. No credential, header,
or full request body is stored in this task package.

## Local Claude Code checks

- [x] Claude Code version is 2.1.220.
- [x] Settings schema accepts through `xhigh`, not persisted `max`.
- [x] Default `yolo-mnt` launch records `effort=max` through the wrapper flag.
- [x] Explicit trailing `--effort xhigh` records `effort=xhigh`.
- [x] `CLAUDE_CODE_EFFORT_LEVEL` is absent.
- [x] `alwaysThinkingEnabled` remains true.
- [x] Hidden `--thinking` validation accepts only `enabled`, `adaptive`, and
  `disabled`; parser checks used a loopback-only dummy endpoint.
- [x] No permanent `--thinking disabled` or `--thinking adaptive` override was
  added.
- [x] `DISABLE_INTERLEAVED_THINKING=1` is DeepSeek-profile-only.
- [x] A real image `Read` is denied by the hook; no image block reaches the API.
- [x] The guarded Claude session continues successfully after the denial.

## Repository gates

- [x] `gofmt -w .`
- [x] `go test ./internal/runtime/executor/helps ./internal/runtime/executor`
- [x] `go test ./internal/runtime/executor/helps ./internal/runtime/executor ./sdk/api/handlers/claude`
- [x] `go test ./...`
- [x] Focused `go test -race` for the changed helper, executor, and Claude error paths
- [x] `go build -o test-output ./cmd/server && rm test-output`
- [x] `git diff --check`
- [x] No changes under `internal/translator/**`
- [x] No credential patterns in changed or newly added repository files
- [x] Image and named-tool-choice validation makes zero upstream attempts and returns
  a request-scoped structured 400
- [x] Valid upstream JSON retains its error type/message and a safe body/header
  request ID in the Claude-compatible error envelope

## Deployment boundary

- [x] Production config hot reload confirmed `bootstrap-retries: 1` in both host
  and current container views.
- [x] Production container remained running on `sha-d6027f...`.
- [ ] Exact Claude-thinking restoration is live in production; pending an explicit
  commit/build/deploy action.
