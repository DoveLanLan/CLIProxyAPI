# hewei journal 9

- Date: 2026-05-22
- Title: Merge upstream non-Docker changes
- Commit:

## Summary

Conclusions/decisions: merged `upstream/main` from `router-for-me/CLIProxyAPI` at `21fad9dbb447a2ab70d51d0ac3e3d032525a6054` while excluding `.github` and Docker-related paths. The merge keeps local workflow state and local fork behavior where it conflicts with upstream.

What changed: imported upstream v7 module path and broad upstream updates across Home support, xAI support, Codex client model handling, image/video handlers, Redis queue protocol support, runtime executors, registry data, thinking metadata, management APIs, SDK handlers, and protocol translators. Preserved CPA-Manager panel defaults, OpenAI-compatible `xhigh` thinking defaults, DeepSeek registry helpers, Codex free-tier `gpt-5.5` filtering, and Codex invalidated OAuth token fallback.

Verification: `gofmt` on changed Go files; `git diff --check`; no unresolved conflict files or conflict markers; excluded `.github`/Docker paths unchanged; focused Codex invalidated-token fallback test; `go test ./sdk/cliproxy/auth`; `go test ./...`; and `go build -o test-output ./cmd/server && rm test-output`. All passed.

Risks/rollback: this is a large upstream sync and should be rolled out carefully despite passing tests. Rollback is a straight revert of the sync commit, or discard the squash merge before commit; no data migration rollback is required.
