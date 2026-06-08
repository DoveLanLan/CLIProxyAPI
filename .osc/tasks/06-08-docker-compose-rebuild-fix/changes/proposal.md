# Proposal: Docker Compose Rebuild Fix

- Date: 2026-06-08
- Owner(s): Codex
- Stakeholders: CLIProxyAPI local Docker users
- Status: Accepted

## Context / Problem
`docker compose build cli-proxy-api` now reaches dependency download, but the builder image cannot resolve a Git-backed module because `git` is missing from the Alpine-based Go image.

## Goals (Why/What)
- Make the current Dockerfile build successfully with Docker Compose.
- Keep the runtime image unchanged and minimal.

## Constraints
- Keep the change scoped to Docker build dependencies.
- Do not change runtime ports, volumes, config paths, or application behavior.

## Non-goals
- Do not investigate the management page `invalid_yaml` behavior in this change.
- Do not refactor Docker Compose or application code.

## Proposed Approach (high-level)
Install `git` in the builder stage before `go mod download` so Go can resolve modules that require VCS access. The final Alpine runtime stage remains unchanged.

## Risks & Mitigations
- Risk: Build dependencies increase the builder stage footprint.
  - Mitigation: Install `git` only in the builder stage; it is not copied into the runtime image.

## Open Questions (max 3)
- None.
