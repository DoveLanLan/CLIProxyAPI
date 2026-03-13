# Change Summary: Configure Custom Providers And Docker Compose Startup

- Date: 2026-03-09
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`, `tasks.md`

## What changed

- Created a local gitignored `config.yaml` with:
  - a generated downstream proxy API key
  - one `openai-compatibility` provider for the custom `nih.cc` upstream
  - one custom `codex-api-key` provider for the `codex.zcaoyao...` upstream
  - three custom `claude-api-key` providers for the supplied Claude-compatible upstreams, including SiliconFlow
- Queried upstream model listings for the OpenAI-style providers and wrote model aliases into the runtime config so those providers are immediately usable.
- Added a local `docker-compose.local.yml` override that replaces the base compose port list with only `8317`, because host port `8085` was already occupied on this machine.
- Updated the tracked `docker-compose.yml` service definition to map `host.docker.internal` to Docker's `host-gateway` for `cli-proxy-api`.
- Corrected the live `config.yaml` so the `claude-local` upstream now points to `http://host.docker.internal:8990` instead of `http://localhost:8990`.
- Started the service with Docker Compose and verified both the root endpoint and `/v1/models`.
- Recreated `cli-proxy-api` and verified the container can now reach `http://host.docker.internal:8990/v1/messages`, which returns an upstream HTTP response instead of a connection failure.

## Why

The user asked for a working local runtime configuration and a compose-based startup flow using supplied custom provider credentials, plus guidance for official Codex Team account setup. The implemented configuration keeps custom API-key providers separate from official OAuth-backed Codex Team accounts and makes the service usable immediately on the local machine.

This follow-up specifically fixes the Docker networking mismatch where `localhost:8990` inside the proxy container incorrectly pointed back to the container itself rather than to the host-published upstream service.

## Notable decisions

- Used `codex-api-key` for the custom Codex-style upstream instead of treating it as a generic OpenAI-compat provider.
- Set `force-model-prefix: true` so custom providers are only used when explicitly addressed via prefix.
- Translated the host-local Claude endpoint to `host.docker.internal` for container reachability.
- For the SiliconFlow Claude-compatible endpoint, used a clean alias `minimax-m2.5` instead of exposing the upstream model name with embedded slashes.
