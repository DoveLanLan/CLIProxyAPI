# Spec: Configure Custom Providers And Docker Compose Startup

- Date: 2026-03-09
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## Repo Snapshot (from step 0)

- Modules/components:
  - `cmd/server` runnable server entrypoint
  - `internal/api`, `internal/config`, `internal/store`, `internal/logging` runtime/config surfaces
  - `sdk/cliproxy` shared lifecycle/runtime layer
- Toolchains:
  - Build: Go modules, `go build ./cmd/server`, Dockerfile, `docker-compose.yml`
  - CI: PR build job and translator path guard
  - Runtime config: `config.example.yaml`, local `config.yaml` volume-mounted into the container
- Confidence: High
- Evidence: `cmd/server/main.go`, `config.example.yaml`, `docker-compose.yml`, `.github/workflows/pr-test-build.yml`, `.github/workflows/pr-path-guard.yml`

## Scope

### In scope

- Create a task-scoped change package for this runtime configuration change.
- Create a local `config.yaml` with:
  - top-level proxy API key(s)
  - `openai-compatibility` entry for the custom OpenAI-style upstream
  - `codex-api-key` entry for the custom Codex-style upstream
  - `claude-api-key` entries for the custom Claude-style upstreams
- Query upstream model lists where necessary to make the config usable.
- Start the service with `docker compose`.
- Update Docker host resolution for the host-local Claude-compatible upstream.
- Provide Codex Team setup guidance for official OAuth/device-login accounts.

### Out of scope

- Editing provider executor code or translators.
- Moving the target upstream service into the same user-defined Docker network.
- Managing official Claude/Codex/Gemini OAuth accounts in this step.

## Acceptance Criteria (testable)

1. A gitignored local runtime config exists at `config.yaml` and parses under the repository’s config schema. (Verify: inspect file + successful container startup)
2. The config contains the user-supplied custom upstreams using repo-supported sections (`openai-compatibility`, `codex-api-key`, `claude-api-key`). (Verify: inspect `config.yaml`)
3. `docker compose up -d` starts a `cli-proxy-api` container that reaches the running state. (Verify: `docker compose ps`)
4. The proxy responds on the configured port after startup. (Verify: `curl http://localhost:8317/`)
5. The compose service exposes `host.docker.internal` inside `cli-proxy-api`, and the host-local Claude upstream is reachable through `http://host.docker.internal:8990/v1/messages`. (Verify: `docker exec cli-proxy-api wget ... http://host.docker.internal:8990/v1/messages` returns an upstream HTTP status instead of connection failure)
6. The user receives explicit instructions for configuring official Codex Team accounts via the repo’s Codex login flow rather than via `config.yaml`. (Verify: final response)

## Behavior / Requirements

- The runtime config must remain outside version control and use the repository’s expected `config.yaml` mount path.
- A dedicated local proxy API key should be generated for downstream clients instead of reusing upstream provider keys as the proxy’s own auth secret.
- For `openai-compatibility`, provider aliases should be based on actual upstream model listings when possible, because alias-based routing is how this repo resolves those providers.
- For `codex-api-key` and `claude-api-key`, empty `models:` lists are acceptable because the runtime falls back to built-in model registries, but exact upstream model lists may still be populated if cheaply available.
- For the host-local Claude upstream, prefer `host.docker.internal` plus Docker host-gateway mapping over cross-project network rewiring so existing local proxy/network setups remain unchanged.
- Official Codex Team accounts should be stored as auth files in `auths/` through built-in Codex login commands, not hardcoded into `config.yaml`.

## Edge Cases

- Upstream `/models` endpoint may be unavailable or require a nonstandard header shape.
- One custom upstream may return a subset of models, while another uses standard model defaults.
- Docker may be installed but the daemon may not be running.
- Existing local `auths/`, `logs/`, or stale containers may affect startup.
- If the config contains unreachable local-only upstreams such as `http://127.0.0.1:8990`, the server should still start even if those upstreams are not currently healthy.
- Linux Docker environments may not resolve `host.docker.internal` unless `extra_hosts` explicitly adds `host-gateway`.

## Compatibility Notes

- Backwards compatibility: no tracked source-code changes are intended; runtime behavior changes are confined to local `config.yaml`.
- Data/migrations: none; auth files remain file-backed under the mounted `auths/` directory.
- Config/flags: `config.yaml` is mounted by `docker-compose.yml`; this change also adds an `extra_hosts` mapping for host-gateway resolution. Official Codex Team setup later uses `--codex-device-login` or `--codex-login` against the same config/auth mount.

## API/UX Decisions (if applicable)

- Inputs/outputs: downstream clients authenticate with a local proxy API key defined in `config.yaml`; upstream credentials remain provider-specific.
- States/errors: container startup and health are checked via compose status and root endpoint response.
- Telemetry/logging: no special logging changes planned; use default compose/container logs if startup fails.
- Accessibility/i18n (if UI): not applicable.
