# Tasks: Configure Custom Providers And Docker Compose Startup

- Date: 2026-03-09
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`

## Assumptions

- Docker and Docker Compose are available on this machine.
- The supplied provider keys are intended for immediate local runtime configuration.
- Creating a gitignored `config.yaml` is acceptable and preferred over editing tracked example docs.

## Checklist

- [x] 1) Prepare local runtime configuration
  - Target: `config.yaml`
  - Change: create a gitignored runtime config with a generated local proxy API key plus the user-supplied custom provider entries, using repo-supported config sections
  - Verify: config file exists and matches the config schema shape

- [x] 2) Populate provider model metadata where needed
  - Target: user-supplied upstream endpoints, `config.yaml`
  - Change: query provider model endpoints where necessary, especially for `openai-compatibility`, and write usable alias mappings
  - Verify: resulting config contains non-empty aliases for the generic OpenAI-compatible provider

- [x] 3) Start the stack with Docker Compose
  - Target: `docker-compose.yml`, local Docker runtime
  - Change: launch the service using the existing compose definition plus a local override file because host port `8085` was already occupied
  - Verify: `docker compose ps` shows the container running and `curl http://localhost:8317/` responds successfully

- [x] 4) Document operational follow-up
  - Target: final response, `.osc/quality-gate.md`
  - Change: explain how official Codex Team accounts should be added through the built-in Codex login flow rather than through API-key config
  - Verify: user has concrete commands and auth-file location guidance

- [x] 5) Fix Docker-to-host routing for the local Claude upstream
  - Target: `docker-compose.yml`, `config.yaml`
  - Change: add a `host.docker.internal` host-gateway mapping for `cli-proxy-api` and point the `claude-local` base URL at `http://host.docker.internal:8990`
  - Verify: recreate `cli-proxy-api`, then confirm `http://host.docker.internal:8990/v1/messages` is reachable from inside the container

## Notes

- Discovered runtime issue: base `docker-compose.yml` publishes several host ports and failed on `0.0.0.0:8085` because that port is already allocated locally.
- Mitigation used: added `docker-compose.local.yml` with a `!override` `ports` section that only publishes `8317`.
- For container networking, the user-supplied Claude endpoint `http://127.0.0.1:8990` was translated to `http://host.docker.internal:8990` in `config.yaml`; inside Docker, `127.0.0.1` would have pointed at the container itself.
- Follow-up issue on 2026-03-13: the live `config.yaml` had regressed to `http://localhost:8990`, and the compose service was still missing the host-gateway alias, so the container could not reach the host-published upstream until both settings were restored together.
- Added an additional Claude-compatible provider for SiliconFlow with prefix `claude-sf` and explicit model alias `minimax-m2.5`, verified through `/v1/models`.
