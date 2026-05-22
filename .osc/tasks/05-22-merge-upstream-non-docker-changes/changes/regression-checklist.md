# Regression Checklist: Merge Upstream Non-Docker Changes

- Date: 2026-05-22
- Related: proposal.md, spec.md, tasks.md

## Automated Checks

- [x] No unresolved merge files: `git diff --name-only --diff-filter=U`
- [x] No conflict markers: `rg -n '^(<<<<<<<|=======|>>>>>>>)' -S . --glob '!logs/**' --glob '!auths/**' --glob '!tmp/**'`
- [x] Excluded files unchanged: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example`
- [x] Formatting: `gofmt` on changed Go files
- [x] Whitespace check: `git diff --check`
- [x] Focused Codex invalidated-token fallback: `go test ./sdk/cliproxy/auth -run TestManager_CodexInvalidatedOAuthTokenDisablesAndFallsBackWithMaxRetryOne -v`
- [x] Focused auth package: `go test ./sdk/cliproxy/auth`
- [x] Full test suite: `go test ./...`
- [x] Server compile gate: `go build -o test-output ./cmd/server && rm test-output`

## Manual Checks

- [x] Confirmed `.github` and Docker-related upstream changes were excluded.
- [x] Confirmed translator changes are part of the broader upstream sync, not a translator-only task.
- [x] Confirmed the local CPA-Manager repository default remains `seakee/CPA-Manager`.
- [x] Confirmed local OpenAI-compatible `xhigh` thinking defaults and tests remain present after the upstream v7 module path upgrade.
- [x] Confirmed local Codex invalidated OAuth token fallback still passes with `max-retry-credentials: 1`.

## Residual Risk

- This is a large upstream sync touching runtime routing, translators, SDK handlers, registry data, and management APIs. Automated tests and build pass, but production config should still be reviewed before rollout because upstream introduced new config fields and provider behavior.
