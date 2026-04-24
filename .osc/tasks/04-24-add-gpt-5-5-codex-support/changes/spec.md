# Spec: Add GPT-5.5 Codex Model Support

- Date: 2026-04-24
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## Repo Snapshot (from step 0)

- Modules/components:
  - `cmd/server` server entrypoint
  - `internal/registry` static and dynamic model registry
  - `internal/runtime` executor/runtime behavior
  - `test/` top-level regressions
- Toolchains:
  - Build: `go build ./cmd/server`
  - Test: `go test ./...` with focused package tests when possible
  - Quality/CI: PR build workflow plus path guard for `internal/translator/**`
- Confidence: High for build/path guard, Medium for targeted test expectations
- Evidence:
  - `.osc/spec/project-spec.md`
  - `.github/workflows/pr-test-build.yml`
  - `.github/workflows/pr-path-guard.yml`
  - `internal/registry/model_definitions.go`
  - `internal/registry/model_definitions_static_data.go`

## Scope

### In scope

- Add a static OpenAI/Codex model definition for `gpt-5.5`.
- Make `LookupStaticModelInfo("gpt-5.5")` return the new metadata.
- Add regression tests covering the new catalog entry and key metadata fields.

### Out of scope

- Full upstream release synchronization.
- Provider auth changes, runtime routing changes, or payload translation changes.
- New config examples or README updates.

## Acceptance Criteria (testable)

1. `internal/registry.GetOpenAIModels()` includes a `gpt-5.5` model entry. (Verify: focused `go test ./internal/registry`)
2. `internal/registry.LookupStaticModelInfo("gpt-5.5")` returns non-nil metadata with upstream-aligned values for version, display name, context length, max completion tokens, supported parameters, and thinking levels. (Verify: focused `go test ./internal/registry`)
3. The project still builds from the production entrypoint after the catalog update. (Verify: `go build ./cmd/server`)

## Behavior / Requirements

- The new model must be additive and must not remove or rename existing OpenAI/Codex models.
- The new entry must use the same metadata values as upstream `router-for-me/CLIProxyAPI` release `v6.9.36` for:
  - `id`: `gpt-5.5`
  - `version`: `gpt-5.5`
  - `display_name`: `GPT 5.5`
  - `description`: `Frontier model for complex coding, research, and real-world work.`
  - `context_length`: `272000`
  - `max_completion_tokens`: `128000`
  - `supported_parameters`: `["tools"]`
  - thinking levels: `low`, `medium`, `high`, `xhigh`
- The change must fit the local fork's current static-data layout; introducing a new registry source format is not required.

## Edge Cases

- The local fork does not contain upstream's `internal/registry/models/models.json`, so the implementation must not depend on that file existing.
- Static lookup should still return cloned data and should not regress existing clone-safety behavior.
- `gpt-5.5` should not accidentally inherit unsupported `none` reasoning level from older GPT-5.x entries.

## Compatibility Notes

- Backwards compatibility: additive only; existing model IDs remain unchanged.
- Data/migrations: none.
- Config/flags: none.

## API/UX Decisions (if applicable)

- Inputs/outputs: no API shape changes; the model may now appear in model lists backed by static registry metadata.
- States/errors: no new error flow introduced.
- Telemetry/logging: unchanged.
- Accessibility/i18n (if UI): not applicable.
