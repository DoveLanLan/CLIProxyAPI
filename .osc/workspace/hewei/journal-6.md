# hewei journal 6

- Date: 2026-04-24
- Title: Add GPT-5.5 Codex model support
- Commit:

## Summary

Conclusions/decisions: synced the local fork with the upstream `router-for-me/CLIProxyAPI` `v6.9.36` GPT-5.5 support at the level that actually matters in this branch, namely static model metadata and regression coverage. The local fork does not contain upstream's `internal/registry/models/models.json`, so the change was adapted directly into `internal/registry/model_definitions_static_data.go` instead of reshaping the registry source layout.

What changed: added a `gpt-5.5` static OpenAI/Codex model entry with upstream-aligned metadata and added `internal/registry/model_definitions_test.go` to lock the new model's lookup and field values.

Verification: ran `go test ./internal/registry`, `go build ./cmd/server`, and `go build -o test-output ./cmd/server`; all passed, and temporary build outputs were removed.

Risks/rollback: this is metadata-only support, so the residual risk is provider-side access mismatch if a deployment advertises `gpt-5.5` without actual upstream entitlement. Rollback is a straight revert of the `gpt-5.5` static entry and its regression test, followed by rerunning the registry test/build gates.
