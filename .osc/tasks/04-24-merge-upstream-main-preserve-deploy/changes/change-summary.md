# Change Summary: Merge Upstream Main While Preserving Deployment Files

- Date: 2026-04-24
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`, `tasks.md`

## What changed

- Merged `upstream/main` (`f1ba6151`) into local `main`.
- Preserved local workflow/deployment files by restoring `.github/**`, `Dockerfile`, `.dockerignore`, `docker-compose*.yml`, `docker-build.sh`, `.goreleaser.yml`, and `deploy/**` from pre-merge `HEAD`.
- Accepted upstream functional changes, including registry JSON/updater migration, Codex/Responses/image-route work, auth scheduling changes, Antigravity/Claude updates, and Qwen/iFlow removal.
- Added a small compatibility helper `persistWith` back to management `Handler` because local `config_basic.go` still uses it after adopting upstream `handler.go`.
- Adjusted the GPT-5.5 registry test to match upstream's current catalog tiers: GPT-5.5 is asserted for team/plus/pro and static lookup, not free.

## Why

The fork needed to catch up with upstream functional code while keeping local production deployment and workflow files stable. The user explicitly accepted dropping Qwen and iFlow.

## Notable decisions

- Upstream functional conflicts were resolved in favor of upstream.
- Protected deployment/workflow paths were resolved in favor of local `HEAD`.
- The merge was completed locally; remote push and deployment remain separate operations.
