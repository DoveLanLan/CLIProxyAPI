# Assumptions

- The bytevirt error is representative of a client-generated array schema with no `items` field.
- The shared sanitizer is the correct boundary because both Claude and Gemini Antigravity requests use it.

# Checklist

- [x] Add array-items fallback in the shared schema sanitizer.
  - Target: `internal/util/gemini_schema.go`
  - Verify: focused utility tests.
- [x] Add regression coverage for top-level and nested missing-items arrays.
  - Target: `internal/util/gemini_schema_test.go`
  - Verify: `go test ./internal/util`.
- [x] Run repository format, tests, and required build.
  - Target: repository quality gates
  - Verify: `gofmt`, focused tests, `go test ./...`, and `go build -o test-output ./cmd/server`.
- [x] Record closure artifacts and quality results.
  - Target: `.osc/tasks/08-20-fix-antigravity-claude-tool-schema/changes/`, `.osc/quality-gate.md`
  - Verify: sanitized diff and command outcomes recorded.
