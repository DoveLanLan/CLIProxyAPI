# Change Summary: Merge Upstream Non-Docker Changes

- Date: 2026-05-22
- Related: proposal.md, spec.md, tasks.md
- Upstream: `upstream/main` at `21fad9dbb447a2ab70d51d0ac3e3d032525a6054`
- Merge base: `f1ba6151a99240902bcda12102c921b0ead01d2d`

## Summary

Merged the current upstream `router-for-me/CLIProxyAPI` main branch into the local working tree with `.github` and Docker-related files excluded. The sync imports upstream's v7 module path, Home control plane support, xAI provider support, Codex client model registry/handlers, OpenAI image/video handler updates, Redis queue protocol handling, logging and usage improvements, thinking/reasoning updates, and broad translator/runtime fixes.

## Local Behavior Preserved

- `.github/**`, `Dockerfile`, `.dockerignore`, `docker-build.*`, `docker-compose*.yml`, and `.env.cluster.example` were kept unchanged.
- Local `.osc` workflow state was preserved.
- Local CPA-Manager default panel repository remains `https://github.com/seakee/CPA-Manager`.
- OpenAI-compatible default thinking still allows `none`, `low`, `medium`, `high`, and `xhigh`, including zero thinking budget.
- DeepSeek model registry helpers remain available.
- Codex free-tier model list still filters out local `gpt-5.5` exposure.
- Codex invalidated OAuth token handling still disables the bad auth and falls through to another auth, including when `max-retry-credentials` is `1`.

## Conflict/Compatibility Fixes

- Removed duplicate local tests that upstream had moved or replaced.
- Updated local test imports for the upstream module path change to `/v7`.
- Restored local Codex invalidated-token state handling around upstream's newer request preparation and Home retry paths.
- Kept the upstream implementation for new runtime, SDK, translator, and management behavior where it compiled and passed tests.

## Validation

- PASS: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example`
- PASS: `git diff --name-only --diff-filter=U`
- PASS: `rg -n '^(<<<<<<<|=======|>>>>>>>)' -S . --glob '!logs/**' --glob '!auths/**' --glob '!tmp/**'`
- PASS: `gofmt` on changed Go files
- PASS: `git diff --check`
- PASS: `go test ./sdk/cliproxy/auth -run TestManager_CodexInvalidatedOAuthTokenDisablesAndFallsBackWithMaxRetryOne -v`
- PASS: `go test ./sdk/cliproxy/auth`
- PASS: `go test ./...`
- PASS: `go build -o test-output ./cmd/server && rm test-output`
