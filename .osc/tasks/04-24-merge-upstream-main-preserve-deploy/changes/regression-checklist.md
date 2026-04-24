# Regression Checklist: Merge Upstream Main While Preserving Deployment Files

- Date: 2026-04-24
- Related: `spec.md`, `tasks.md`

## Gates (from Repo Snapshot)

- Build: `go build -o test-output ./cmd/server`
- Tests: `go test ./...`
- Diff checks: `git diff --check`
- Protected paths: `git diff --name-status HEAD -- .github Dockerfile .dockerignore docker-compose.yml docker-compose.*.yml docker-build.sh .goreleaser.yml deploy`
- Provider removal check: Qwen/iFlow paths absent

## Executed

- [x] `git diff --name-only --diff-filter=U`
- [x] `git diff --name-status HEAD -- .github Dockerfile .dockerignore docker-compose.yml docker-compose.*.yml docker-build.sh .goreleaser.yml deploy`
- [x] `git diff --check`
- [x] `go build -o test-output ./cmd/server`
- [x] `go test ./internal/registry ./internal/api/handlers/management ./sdk/cliproxy/auth ./sdk/api/handlers/openai ./internal/runtime/executor`
- [x] `go test ./...`

## Manual checks

- [x] Confirmed Qwen/IFlow paths are absent after merge.
- [x] Confirmed protected workflow/Docker/deploy paths have no diff versus pre-merge `HEAD`.

## Follow-up regression areas

- Exercise Codex `/v1/responses`, `/v1/chat/completions`, websocket, and image-generation paths against production credentials before deployment.
- Exercise management auth file upload/delete/list flows in the management UI.
- Confirm production config no longer references Qwen/IFlow providers before deploying this branch.
