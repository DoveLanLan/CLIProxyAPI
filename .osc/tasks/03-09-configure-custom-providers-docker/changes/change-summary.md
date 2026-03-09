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
- Started the service with Docker Compose and verified both the root endpoint and `/v1/models`.

## Why

The user asked for a working local runtime configuration and a compose-based startup flow using supplied custom provider credentials, plus guidance for official Codex Team account setup. The implemented configuration keeps custom API-key providers separate from official OAuth-backed Codex Team accounts and makes the service usable immediately on the local machine.

## Notable decisions

- Used `codex-api-key` for the custom Codex-style upstream instead of treating it as a generic OpenAI-compat provider.
- Set `force-model-prefix: true` so custom providers are only used when explicitly addressed via prefix.
- Translated the host-local Claude endpoint to `host.docker.internal` for container reachability.
- For the SiliconFlow Claude-compatible endpoint, used a clean alias `minimax-m2.5` instead of exposing the upstream model name with embedded slashes.
