# Spec: Fix OpenAI Compat XHigh Thinking Preservation

- Date: 2026-05-12
- Owner(s): hewei / Codex
- Related: proposal.md, tasks.md

## Repo Snapshot (from step 0)

- Modules/components: `cmd/server` binary; `internal/thinking` canonical reasoning pipeline; `internal/registry` model metadata; `sdk/cliproxy` runtime/model registration; `test` cross-module regressions.
- Toolchains: Go modules; build via `go build -o test-output ./cmd/server`; tests via `go test ./...`; formatting via `gofmt`.
- Quality/CI: PR build compiles `./cmd/server`; `internal/translator/**` is path-guarded.
- Confidence: High.
- Evidence: `AGENTS.md`, `go.mod`, `.github/workflows/pr-test-build.yml`, `.github/workflows/pr-path-guard.yml`, `.osc/spec/project-spec.md`.

## Scope

### In scope

- Update OpenAI-compatible model registration defaults for nil `thinking` metadata.
- Preserve explicit `thinking` metadata from config.
- Reuse static level-based thinking metadata where safe for OpenAI-compatible requests.
- Add regression tests for namespaced DeepSeek xhigh preservation.
- Update config example comments to match the default.

### Out of scope

- Translator changes.
- Remote model catalog changes.
- Local deployment secret/config edits.

## Acceptance Criteria (testable)

1. A configured OpenAI-compatible model named `deepseek-ai/deepseek-v3.2` without explicit `thinking` preserves suffix budget `64000` as `reasoning_effort="xhigh"`. (Verify: focused Go test)
2. Explicit `thinking` metadata in config remains authoritative. (Verify: focused Go test)
3. `config.example.yaml` documents the new default accurately. (Verify: review diff)
4. The server still compiles. (Verify: `go build -o test-output ./cmd/server && rm test-output`)

## Behavior / Requirements

OpenAI-compatible configured models with omitted `thinking` metadata should default to a permissive level list: `none`, `low`, `medium`, `high`, and `xhigh`, with zero thinking allowed. If config supplies `thinking`, use it unchanged. If a static model lookup provides discrete levels, copy those levels instead of using the generic default.

## Edge Cases

- Namespaced upstream IDs such as `deepseek-ai/deepseek-v3.2` may not exist in static metadata; generic OpenAI-compatible defaults still apply.
- Static metadata with budget-only thinking is not reused for OpenAI-compatible reasoning fields.
- Duplicate aliases still deduplicate as before.
- Existing hot-reload hashing continues to include explicit thinking metadata.

## Compatibility Notes

- Backwards compatibility: normal requests without thinking config are unchanged. Requests that ask for `xhigh` are now forwarded instead of being lowered to `high`.
- Data/migrations: none.
- Config/flags: no new config fields; only the documented default changes.

## API/UX Decisions (if applicable)

- Inputs/outputs: `reasoning_effort` may now be `xhigh` for OpenAI-compatible configured models using default thinking metadata.
- States/errors: upstream services remain responsible for rejecting unsupported compatible-provider extensions.
- Telemetry/logging: existing thinking pipeline debug logs remain sufficient.
