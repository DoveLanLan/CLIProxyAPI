# Proposal: Add GPT-5.5 Codex Model Support

- Date: 2026-04-24
- Owner(s): hewei
- Stakeholders: local fork maintainers, Codex/OpenAI-model users
- Status: Accepted

## Context / Problem

The upstream `router-for-me/CLIProxyAPI` release `v6.9.36` added GPT-5.5 Codex model support. This local fork still stops its built-in OpenAI/Codex static catalog at `gpt-5.4`, so built-in model discovery and static lookup cannot expose `gpt-5.5`.

## Goals (Why/What)

- Add the upstream GPT-5.5 model metadata to the local static OpenAI/Codex registry.
- Add regression coverage so future catalog edits do not silently drop GPT-5.5 support.

## Constraints

- Follow the `osc` artifact-first workflow before editing non-`.osc/` files.
- Keep the change minimal and additive; do not bundle unrelated upstream release sync.
- Preserve current registry behavior and keep `go build ./cmd/server` passing.
- Avoid changes under protected paths such as `internal/translator/**`.

## Non-goals

- Sync the entire upstream `v6.9.36` release into this fork.
- Introduce the upstream `models.json` source-file layout used in another branch of the project.
- Change request-routing, auth flow, or Codex image-generation behavior.

## Proposed Approach (high-level)

Patch the local static registry source at `internal/registry/model_definitions_static_data.go` to add the `gpt-5.5` model entry with the same metadata used upstream, then add a focused `internal/registry` test that validates both the catalog list and static lookup path can see the new model.

## Risks & Mitigations

- Risk: incorrect model metadata could expose the wrong context window or thinking levels.
  - Mitigation: copy the upstream `v6.9.36` GPT-5.5 fields exactly and assert them in tests.
- Risk: the local fork differs structurally from upstream, so a direct cherry-pick may not apply.
  - Mitigation: adapt the upstream change to this fork's existing static-data layout instead of porting file structure.
- Risk: an additive catalog change could accidentally affect other registry behavior.
  - Mitigation: keep the patch limited to one model entry plus targeted tests and run the package tests/build gate.

## Open Questions (max 3)

- None.
