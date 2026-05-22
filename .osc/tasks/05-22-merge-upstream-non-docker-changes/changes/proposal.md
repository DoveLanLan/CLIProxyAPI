# Proposal: Merge Upstream Non-Docker Changes

- Date: 2026-05-22
- Owner(s): hewei / Codex
- Stakeholders: CLIProxyAPI maintainers and local fork users
- Status: Accepted

## Context / Problem

The local `main` branch has diverged from `upstream/main` at `router-for-me/CLIProxyAPI`. The user requested bringing in upstream code changes that are missing locally while excluding `.github` and Docker-related content.

## Goals (Why/What)

- Fetch and compare `upstream/main` against the current branch.
- Merge upstream non-Docker, non-`.github` changes into the local working tree.
- Preserve local-only workflow files and deployment/Docker customizations.
- Verify the resulting Go code with formatting, tests, build, and conflict checks.

## Constraints

- Exclude `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.*`, `docker-compose*.yml`, and the upstream Docker cluster env example `.env.cluster.example` from the applied upstream patch.
- Do not apply a plain `git merge` because upstream lacks local `.osc` task/workflow files and would delete local workflow state.
- Keep Go source formatted with `gofmt`.
- Keep `go build -o test-output ./cmd/server && rm test-output` passing.
- Treat `internal/translator/**` as protected but allowable here because the upstream sync spans broader protocol/runtime changes, not translator-only work.

## Non-goals

- Do not merge upstream CI workflow changes.
- Do not merge Docker image, compose, or Docker build helper changes.
- Do not change local runtime config files such as `config.yaml` or auth material.
- Do not create a merge commit unless explicitly requested.

## Proposed Approach (high-level)

Use the merge base between `HEAD` and `upstream/main` to generate an upstream-only patch, excluding `.github` and Docker-related paths. Apply that patch to the current working tree with three-way fallback, resolve any conflicts in favor of preserving local fork behavior where required, then run repository quality gates.

## Risks & Mitigations

- Risk: Upstream changes touch hundreds of files and may conflict with local fork features.
  - Mitigation: Use three-way patch application, inspect conflicts, and run full Go tests/build.
- Risk: Excluding Docker files may leave docs or config references to Docker-only upstream behavior.
  - Mitigation: Review changed non-Docker config/docs and keep explicit Docker files untouched.
- Risk: `internal/translator/**` changes may trip protected-path review expectations.
  - Mitigation: Document that translator changes are part of a broader upstream sync and verify with tests.

## Open Questions

- None. The user explicitly scoped the exclusion set and requested implementation.
