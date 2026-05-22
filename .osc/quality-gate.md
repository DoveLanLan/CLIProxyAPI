# Quality Gate

- Date: 2026-05-22
- Scope: Make CPA-Manager image configurable for production deployment.

## Assumptions:

- This change is deployment-only and does not alter Go runtime behavior.
- The VPS production stack uses `deploy/compose.production.yml`.

## Suspected Change Scope:

- `deploy/compose.production.yml`
- `deploy/README.md`
- `.osc/tasks/05-22-configurable-cpa-manager-image/**`

## Detected Gates:

- Gate Name: PR Go build
  - Confidence: High
  - Evidence: `.github/workflows/pr-test-build.yml` job `build`, step `Build`, runs `go build -o test-output ./cmd/server`.
- Gate Name: Docker image build/publish
  - Confidence: Medium
  - Evidence: `.github/workflows/docker-image.yml` job `build-and-push` builds CLIProxyAPI image on pushes to `main` and tags.
- Gate Name: Compose render validation
  - Confidence: Medium
  - Evidence: deployment change is in `deploy/compose.production.yml`; previous CPA-Manager deployment task used Docker Compose config rendering as the local validation gate.
- Gate Name: Diff whitespace check
  - Confidence: Medium
  - Evidence: repository convention from prior `.osc` regression checklists and safe text/YAML hygiene.

## Suggested Gate Run (Local):

1. `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test docker compose -f deploy/compose.production.yml config`
   - Rationale: verifies the default CPA-Manager image still renders.
2. `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test CPA_MANAGER_IMAGE=ghcr.io/example/cpa-manager:sha-test docker compose -f deploy/compose.production.yml config`
   - Rationale: verifies the fork image override renders.
3. `git diff --check`
   - Rationale: catches whitespace and patch hygiene issues.
4. `go build -o test-output ./cmd/server && rm test-output`
   - Rationale: mirrors the PR build gate from `.github/workflows/pr-test-build.yml`.

## Final Self-Review:

- Security & secrets: no secrets or credentials added; image override is non-secret configuration.
- Edge cases & error handling: invalid image values fail during Docker pull/start, with rollback by unsetting the env var.
- Backward compatibility / migrations: default image, ports, volumes, and data paths remain unchanged; no migration.
- API/contract compatibility: no API or CLI contracts changed.
- Observability: not applicable for deployment image parameterization.
- Config/env changes: added optional `CPA_MANAGER_IMAGE`; docs updated.
- Performance risk: no runtime performance change.
- Rollback plan: unset `CPA_MANAGER_IMAGE` or set it to `seakee/cpa-manager:latest`, then restart CPA-Manager.

## PR-ready checklist:

- [x] Default compose render: `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test docker compose -f deploy/compose.production.yml config`
- [x] Override compose render: `TAILSCALE_BIND_IP=127.0.0.1 CLI_PROXY_IMAGE=test CPA_MANAGER_IMAGE=ghcr.io/example/cpa-manager:sha-test docker compose -f deploy/compose.production.yml config`
- [x] Diff hygiene: `git diff --check`
- [x] Build: `go build -o test-output ./cmd/server && rm test-output`
