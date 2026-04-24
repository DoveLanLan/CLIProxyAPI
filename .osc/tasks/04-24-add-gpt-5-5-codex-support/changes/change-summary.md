# Change Summary: Add GPT-5.5 Codex Model Support

- Date: 2026-04-24
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`, `tasks.md`

## What changed

- Added a `gpt-5.5` static OpenAI/Codex model definition in `internal/registry/model_definitions_static_data.go`.
- Added `internal/registry/model_definitions_test.go` to lock the GPT-5.5 metadata and static lookup behavior.
- Verified the change with focused package tests and the repository's server build gate.

## Why

This local fork was missing the GPT-5.5 support that upstream `router-for-me/CLIProxyAPI` added in release `v6.9.36`. Without the static definition, built-in model metadata lookup could not surface `gpt-5.5`.

## Notable decisions

- Adapted the upstream change to this fork's existing `model_definitions_static_data.go` layout instead of importing upstream's `models.json` source layout, which does not exist in this branch.
