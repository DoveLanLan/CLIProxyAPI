# Quality Gate: Merge Upstream Non-Docker Changes

## Assumptions

- The change scope is a broad upstream sync from `router-for-me/CLIProxyAPI` `upstream/main`.
- `.github` and Docker-related files are intentionally excluded from the merge.
- Translator changes are allowed here because the task is not translator-only and the upstream sync spans runtime, SDK, registry, API, and tests.

## Suspected Change Scope

- Server/runtime: `cmd/server`, `internal/api`, `internal/auth`, `internal/runtime`, `internal/home`, `internal/store`, `internal/watcher`.
- SDK/API: `sdk/api`, `sdk/auth`, `sdk/cliproxy`, `sdk/translator`.
- Registry/thinking/translators: `internal/registry`, `internal/thinking`, `internal/translator`.
- Docs/examples/assets/module metadata: `README*`, `examples/`, `assets/`, `go.mod`, `go.sum`, `config.example.yaml`.

## Detected Gates

- Gate Name: Go format
  - Confidence: High
  - Evidence: `AGENTS.md` requires `gofmt -w .` after Go changes.
- Gate Name: Go tests
  - Confidence: High
  - Evidence: `AGENTS.md` lists `go test ./...`; repository has package and cross-module `*_test.go` coverage.
- Gate Name: Go build
  - Confidence: High
  - Evidence: `AGENTS.md` requires `go build -o test-output ./cmd/server && rm test-output`; project spec mirrors PR build gate.
- Gate Name: Conflict/exclusion checks
  - Confidence: High
  - Evidence: task spec excludes `.github` and Docker-related paths and requires no conflict markers.
- Gate Name: Translator path review
  - Confidence: High
  - Evidence: `AGENTS.md` marks `internal/translator/**` as protected unless part of broader changes.

## Suggested Gate Run (Local)

1. `gofmt` on changed Go files - required after Go edits.
2. `git diff --check` - catches whitespace/conflict formatting issues.
3. `git diff --name-only --diff-filter=U` and `rg -n '^(<<<<<<<|=======|>>>>>>>)' -S . --glob '!logs/**' --glob '!auths/**' --glob '!tmp/**'` - verify conflict resolution.
4. `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example` - verify excluded files remain unchanged.
5. `go test ./...` - validates package and cross-module regressions.
6. `go build -o test-output ./cmd/server && rm test-output` - required server compile gate.
7. `git diff --cached --name-only -- internal/translator` - review translator scope as part of broad upstream sync.

## Actual Gate Results

- PASS: `gofmt` on changed Go files.
- PASS: `git diff --check`.
- PASS: `git diff --name-only --diff-filter=U`.
- PASS: `rg -n '^(<<<<<<<|=======|>>>>>>>)' -S . --glob '!logs/**' --glob '!auths/**' --glob '!tmp/**'`.
- PASS: `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example`.
- PASS: `go test ./sdk/cliproxy/auth -run TestManager_CodexInvalidatedOAuthTokenDisablesAndFallsBackWithMaxRetryOne -v`.
- PASS: `go test ./sdk/cliproxy/auth`.
- PASS: `go test ./...`.
- PASS: `go build -o test-output ./cmd/server && rm test-output`.
- REVIEWED: `git diff --cached --name-only -- internal/translator` shows 92 translator files, accepted as part of the broad upstream sync.

## Final Self-Review

- Security & secrets: no auth material, logs, or runtime `config.yaml` were edited; no secrets were added.
- Edge cases & error handling: preserved local Codex invalidated OAuth token disable-and-fallback behavior with focused regression coverage.
- Backward compatibility / migrations: upstream upgrades the Go module to `/v7`; no database or storage migration was introduced.
- API/contract compatibility: upstream API/SDK surface was accepted broadly; local CPA-Manager defaults and OpenAI-compatible thinking behavior were preserved.
- Observability: upstream logging/usage changes were included; secret masking expectations are still covered by tests.
- Config/env changes: upstream config example changes were merged except Docker-related files; existing local runtime config was untouched.
- Performance risk: broad runtime changes are covered by tests/build, but production rollout should still be staged.
- Rollback plan: revert the upstream-sync commit or discard the squash merge before commit; no separate data rollback is required.

## PR-ready checklist

- [x] `gofmt` on changed Go files.
- [x] `git diff --check`.
- [x] `git diff --name-only --diff-filter=U`.
- [x] `rg -n '^(<<<<<<<|=======|>>>>>>>)' -S . --glob '!logs/**' --glob '!auths/**' --glob '!tmp/**'`.
- [x] `git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example`.
- [x] `go test ./sdk/cliproxy/auth -run TestManager_CodexInvalidatedOAuthTokenDisablesAndFallsBackWithMaxRetryOne -v`.
- [x] `go test ./sdk/cliproxy/auth`.
- [x] `go test ./...`.
- [x] `go build -o test-output ./cmd/server && rm test-output`.
- [x] Translator scope reviewed as broad upstream sync.
