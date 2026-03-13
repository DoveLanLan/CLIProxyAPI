# Proposal: Configure Custom Providers And Docker Compose Startup

- Date: 2026-03-09
- Owner(s): hewei
- Stakeholders: local operator, CLIProxyAPI users on this workstation
- Status: Proposed

## Context / Problem

The repository is present locally, but there is no active runtime configuration file for the user’s custom upstream providers. The user wants CLIProxyAPI configured with custom OpenAI-compatible, Codex-compatible, and Claude-compatible API-key providers, then started with `docker compose`. The user also wants guidance for official Codex Team account setup after the custom-provider configuration is in place.

The current local Claude-compatible entry now points at `http://localhost:8990` inside `config.yaml`. When the proxy runs in Docker, that hostname resolves to the `cli-proxy-api` container itself rather than the host-published `8990` service, so `POST /v1/messages` fails before reaching the intended upstream. The compose service also does not yet map `host.docker.internal` to the Docker host gateway.

## Goals (Why/What)

- Create a usable local `config.yaml` from the repo’s config schema using the user-supplied provider credentials.
- Start the proxy with `docker compose` against that config.
- Keep custom API-key providers clearly separated from official Codex Team OAuth accounts.
- Leave an auditable task package describing the configuration and verification flow.
- Make the local Claude-compatible upstream reachable from inside the running Docker container with the smallest possible networking change.

## Constraints

- Must follow the `osc` change workflow before editing non-`.osc/` files.
- Must avoid tracked source-code changes unless required; `config.yaml` is gitignored and is the intended local runtime artifact.
- Must preserve the repo’s provider-specific configuration model:
  - `openai-compatibility` for generic OpenAI-style upstreams
  - `codex-api-key` for custom Codex/OpenAI key-based upstreams
  - `claude-api-key` for Claude-compatible upstreams
- Should not store secrets in tracked documentation artifacts.
- Should avoid unnecessary Docker network rewiring because this workstation also uses local proxy tooling outside this repository.

## Non-goals

- Adding new runtime features or changing provider executor logic.
- Configuring official Codex Team OAuth accounts directly in this step.
- Building a custom dashboard or enabling unrelated management features.
- Attaching the upstream service to the same user-defined Docker network unless the lighter host-gateway approach proves insufficient.

## Proposed Approach (high-level)

Create a new task-local change package, generate a gitignored `config.yaml` containing the user’s custom provider entries plus a local proxy API key, verify or import upstream model listings where needed for `openai-compatibility`, start the service using `docker compose`, and then explain how official Codex Team accounts should be added through the built-in Codex login flow and persisted in `auths/` rather than in `config.yaml`.

For the host-local Claude-compatible upstream, switch the runtime base URL from `localhost:8990` to `host.docker.internal:8990` and add an `extra_hosts` entry mapping `host.docker.internal` to Docker’s `host-gateway`. This preserves the current upstream deployment while making the host-published port reachable from the proxy container.

## Risks & Mitigations

- Risk: `openai-compatibility` providers are hard to use without model aliases.
  - Mitigation: query upstream `/models` endpoints and populate aliases automatically when possible.
- Risk: container startup may fail due to missing Docker runtime or unreachable upstreams.
  - Mitigation: validate the compose configuration first, then capture container status/logs after startup.
- Risk: mixing official Codex Team accounts with API-key providers could create the wrong auth model.
  - Mitigation: keep official Codex Team guidance separate and route it through `--codex-device-login` / auth-file flows.
- Risk: `host.docker.internal` does not resolve by default on this Linux container setup.
  - Mitigation: declare `extra_hosts: ["host.docker.internal:host-gateway"]` in compose and verify the upstream becomes reachable from inside `cli-proxy-api`.

## Open Questions (max 3)

- None. Proceeding with the assumption that querying upstream `/models` endpoints is allowed because the user explicitly supplied the target base URLs and keys for configuration.
