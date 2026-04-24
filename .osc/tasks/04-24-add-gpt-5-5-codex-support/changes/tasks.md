# Tasks: Add GPT-5.5 Codex Model Support

- Date: 2026-04-24
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`

## Assumptions

- The local fork should mirror the upstream GPT-5.5 metadata but keep its own current registry layout.
- `internal/registry/model_definitions_static_data.go` is the authoritative source file to edit in this fork.

## Checklist

- [x] 1) Add GPT-5.5 static registry metadata
  - Target: `internal/registry/model_definitions_static_data.go`
  - Change: append a `gpt-5.5` model definition to the OpenAI/Codex static catalog using upstream-aligned metadata
  - Verify: `go test ./internal/registry`

- [x] 2) Add regression coverage for GPT-5.5 lookup
  - Target: `internal/registry/model_definitions_test.go`
  - Change: add focused tests for `GetOpenAIModels()` and `LookupStaticModelInfo("gpt-5.5")`
  - Verify: `go test ./internal/registry`

- [x] 3) Run repo gates and record closure artifacts
  - Target: `internal/registry`, `cmd/server`, `.osc/tasks/04-24-add-gpt-5-5-codex-support/changes/`, `.osc/quality-gate.md`
  - Change: run the focused package tests and build gate, then write summary/regression/rollback outputs
  - Verify: `go test ./internal/registry` and `go build ./cmd/server`

## Notes

- Keep this task scoped to GPT-5.5 support only; do not opportunistically sync other missing upstream model entries in the same patch.
- Completed verification in this task:
  - `go test ./internal/registry`
  - `go build ./cmd/server`
  - `go build -o test-output ./cmd/server`
