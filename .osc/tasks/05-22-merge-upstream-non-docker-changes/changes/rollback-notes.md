# Rollback Notes: Merge Upstream Non-Docker Changes

- Date: 2026-05-22
- Related: proposal.md, spec.md, tasks.md

## Rollback Procedure

- Before commit: discard the in-progress squash merge and task-local edits from the working tree.
- After commit: revert the upstream-sync commit as a whole.
- Keep `.github` and Docker-related files out of the rollback decision because they were intentionally not merged.

## Operational Impact

- Rolling back restores the pre-sync local v6 module path and removes upstream Home, xAI, Codex client model, image/video handler, Redis queue protocol, logging/usage, runtime, registry, thinking, and translator changes.
- No database migration or storage migration was introduced by this sync.
- Existing local runtime configuration files such as `config.yaml`, auth material, and logs were not edited.

## Safer Alternative To Full Rollback

- If a specific upstream feature regresses, revert or patch the affected package while keeping the v7 module path consistent across the repository.
- If a new provider path fails operationally, disable or exclude that provider in config first, then patch the targeted executor/handler with a focused regression test.
