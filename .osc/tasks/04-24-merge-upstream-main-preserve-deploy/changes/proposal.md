# Proposal: Merge Upstream Main While Preserving Deployment Files

- Date: 2026-04-24
- Owner(s): hewei
- Stakeholders: fork maintainer, production operator
- Status: Accepted

## Context / Problem

`origin/main` is behind `upstream/main` by hundreds of functional commits. The fork needs upstream's latest server, SDK, auth, model registry, Codex, Responses, and translator changes. The production fork also carries local GitHub workflow, Docker, and VPS deployment files that must remain unchanged.

## Goals (Why/What)

- Merge the latest `upstream/main` functional code into the current fork.
- Do not preserve removed upstream providers Qwen and iFlow.
- Preserve the current fork's `.github/**`, Docker, compose, and deployment files.
- Keep the repository buildable after conflict resolution.

## Constraints

- Preserve current `.github/**` workflow files.
- Preserve current Docker/deployment files: `Dockerfile`, `docker-compose*.yml`, `.dockerignore`, and `deploy/**`.
- Avoid unrelated manual refactors while resolving merge conflicts.
- Respect upstream's removal of Qwen and iFlow.

## Non-goals

- Do not redesign upstream changes during the merge.
- Do not compare or rewrite deployment workflows beyond preserving current fork contents.
- Do not push to remotes automatically.

## Proposed Approach (high-level)

Merge `upstream/main` into the current `main` branch with `--no-commit`, restore protected deployment and workflow paths from `HEAD`, resolve remaining functional conflicts in favor of upstream unless they conflict with fork-specific requirements, then run focused verification and record closure artifacts.

## Risks & Mitigations

- Risk: Functional conflicts may be extensive because upstream changed registry, auth, executor, translator, and SDK internals.
  - Mitigation: Resolve conflicts module-by-module and run the server build gate.
- Risk: Preserving local deployment files may hide upstream operational updates.
  - Mitigation: Explicitly exclude those paths by design and document that operational sync is out of scope.
- Risk: Removing Qwen/iFlow may break users still configured for those providers.
  - Mitigation: This is accepted by the user for this merge.

## Open Questions (max 3)

- None.
