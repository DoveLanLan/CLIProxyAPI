# Rollback Notes: Add GPT-5.5 Codex Model Support

- Date: 2026-04-24
- Related: `spec.md`, `tasks.md`

## Rollback strategy

Revert the commit that adds `gpt-5.5` to `internal/registry/model_definitions_static_data.go` and removes `internal/registry/model_definitions_test.go`, then rerun `go test ./internal/registry` and `go build -o test-output ./cmd/server`.

## Data / migration considerations

- No schema or data migration is involved.
- No persisted config format changes are involved.

## Operational notes

- Monitoring/alerts to watch: requests or UI surfaces that begin selecting `gpt-5.5` after this change.
- Known residual effects: this is metadata support only; if an upstream credential does not actually provide `gpt-5.5`, callers may still need provider-side access to use it successfully.
