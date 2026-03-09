# Proposal: Configure Official Codex Team Account On Home Computer

- Date: 2026-03-09
- Owner(s): hewei
- Stakeholders: home-machine operator
- Status: Proposed

## Context / Problem

The local workstation already has custom provider configuration, but official Codex Team OAuth accounts were intentionally left out of `config.yaml`. The user wants a follow-up task they can execute later on a home computer to add official Codex Team accounts the repo-supported way.

## Goals (Why/What)

- Document the exact steps to add official Codex Team accounts using the built-in Codex login flow.
- Keep official Codex Team OAuth accounts separate from custom `codex-api-key` providers.
- Make the task runnable on another machine that uses the same Docker-based deployment pattern.

## Constraints

- Official Codex Team accounts must not be hardcoded into `config.yaml`.
- Credentials should be stored in `auths/` through the repo’s auth-file workflow.
- The task should assume Docker-based runtime because the current local setup uses Compose + container exec.
- The task must not expose secrets inside tracked task artifacts.

## Non-goals

- Executing the login now.
- Modifying runtime source code.
- Re-documenting the already-completed custom provider setup.

## Proposed Approach (high-level)

Create a standalone follow-up task that assumes the home computer has the repo plus the local runtime files, starts the proxy container, runs `--codex-device-login` inside the container, completes the device-code authorization in a browser, verifies that auth files are written under `auths/`, and confirms that official unprefixed Codex models appear alongside the existing prefixed custom provider models.

## Risks & Mitigations

- Risk: the home machine may not have the same local runtime files (`config.yaml`, `docker-compose.local.yml`).
  - Mitigation: include an explicit prerequisite step to copy or recreate those files securely before login.
- Risk: users may confuse official Codex Team OAuth models with the custom `linuxdo/*` API-key provider.
  - Mitigation: document that official accounts should remain unprefixed while custom API-key providers stay prefixed.
- Risk: browser-based OAuth may be inconvenient on a remote or headless machine.
  - Mitigation: use `--codex-device-login`, which is the more robust terminal-friendly flow.

## Open Questions (max 3)

- None. The repository already exposes `--codex-login` and `--codex-device-login`, and the Docker runtime layout is known.
