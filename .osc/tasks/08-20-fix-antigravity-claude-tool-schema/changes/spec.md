# Scope

The JSON-schema cleanup used before Antigravity tool requests, plus focused utility tests.

# Acceptance Criteria

- A schema containing `properties.cookies: {"type":"array"}` is cleaned into a schema containing a valid `items` object schema.
- Nested array properties receive the same fallback.
- Arrays with explicit `items` are not changed.
- Focused tests and `go build -o test-output ./cmd/server` pass.

# Behavior / Requirements

- The fallback must be valid for Antigravity and Gemini tool declarations.
- The fallback must be permissive and must not introduce required fields.
- The sanitizer must remain deterministic.

# Edge Cases

- Missing or non-string `type` values must not panic.
- Existing `items` values of any valid schema shape must be preserved.
- Empty object schemas still use the existing Antigravity placeholder behavior.

# Compatibility Notes

This is an upstream-compatibility fix for tool schemas. It does not change public routes or configuration.
