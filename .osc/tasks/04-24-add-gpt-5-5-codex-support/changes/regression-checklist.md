# Regression Checklist: Add GPT-5.5 Codex Model Support

- Date: 2026-04-24
- Related: `spec.md`, `tasks.md`

## Gates (from Repo Snapshot)

- Build: `go build -o test-output ./cmd/server`
- Tests: `go test ./internal/registry`
- Lint/format: `gofmt -w internal/registry/model_definitions_test.go internal/registry/model_definitions_static_data.go`
- Other: `git diff --name-only -- internal/translator`

## Manual checks (if applicable)

- If you expose a model listing endpoint backed by static registry metadata, fetch the list and confirm `gpt-5.5` is present. (Expected: `gpt-5.5` appears with display name `GPT 5.5`)
- If you rely on static model lookup in any management or client-facing UI, verify it shows context length `272000` and reasoning levels up to `xhigh`. (Expected: metadata matches upstream values)

## Edge-case re-tests

- Confirm `LookupStaticModelInfo("gpt-5.5")` still returns a populated model after future OpenAI catalog edits.
- Confirm no unrelated OpenAI/Codex model IDs were renamed or removed from `GetOpenAIModels()`.
