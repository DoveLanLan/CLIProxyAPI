# Spec: Configure Official Codex Team Account On Home Computer

- Date: 2026-03-09
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## Repo Snapshot (from step 0)

- Modules/components:
  - `cmd/server` exposes the CLI flags, including Codex login flows
  - `docker-compose.yml` mounts `config.yaml` and `auths/`
  - local runtime currently also uses `docker-compose.local.yml` to publish only `8317`
- Toolchains:
  - Docker Compose for runtime
  - container exec for interactive login flow
  - auth persistence through mounted `auths/`
- Confidence: High
- Evidence: `cmd/server/main.go`, `docker-compose.yml`, `.osc/tasks/03-09-configure-custom-providers-docker/changes/spec.md`

## Scope

### In scope

- Document prerequisites for reproducing the current proxy runtime on a home machine.
- Document the official Codex Team device-login command to run inside the container.
- Document where the resulting OAuth credentials are stored.
- Document how to verify official Codex Team models are active after login.

### Out of scope

- Running the login now.
- Migrating existing auth files between machines automatically.
- Changing custom provider configuration.

## Acceptance Criteria (testable)

1. The task identifies the exact CLI flags supported by the repo for official Codex login. (Verify: `cmd/server/main.go`)
2. The task lists the container command to run device login against the mounted runtime config. (Verify: `tasks.md`)
3. The task explains where auth files are written and how they relate to the mounted `auths/` directory. (Verify: `tasks.md`)
4. The task provides a post-login verification flow for official Codex Team models. (Verify: `tasks.md`)

## Behavior / Requirements

- The home computer should first have the working local runtime files available:
  - `config.yaml`
  - `docker-compose.local.yml` if host-port conflicts exist or only `8317` should be published
- The container must be running before device login is attempted.
- Device login should run inside the container so the generated auth files land in the mounted `auths/` directory on the host.
- Official Codex Team accounts should stay separate from the custom `codex-api-key` section already configured in `config.yaml`.
- Because `force-model-prefix: true` is enabled in the current runtime config, official Codex Team accounts should remain unprefixed and the custom API-key provider should continue using `linuxdo/*`.

## Edge Cases

- Home machine Docker daemon not running.
- `config.yaml` missing or stale on the home machine.
- Device-code authorization not completed before expiry.
- Multiple official Codex Team accounts needed later.
- Existing `auths/` directory permissions or stale credentials causing load conflicts.

## Compatibility Notes

- Backwards compatibility: no code change; this is an operator-run auth onboarding task.
- Data/migrations: auth data is file-backed under `auths/`.
- Config/flags: login uses `--codex-device-login` or optionally `--codex-login`; runtime still points at `/CLIProxyAPI/config.yaml` inside the container.

## API/UX Decisions (if applicable)

- Inputs/outputs: operator runs a container exec command and completes device-code authorization in a browser.
- States/errors: success means new auth files appear and official Codex models become available; failure means no new auth files or no new unprefixed Codex models.
- Telemetry/logging: use container logs and `/v1/models` to confirm account load.
- Accessibility/i18n (if UI): not applicable.
