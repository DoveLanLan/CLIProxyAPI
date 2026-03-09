# Tasks: Configure Official Codex Team Account On Home Computer

- Date: 2026-03-09
- Owner(s): hewei
- Related: `proposal.md`, `spec.md`

## Assumptions

- The home machine will use the same repository layout.
- The home machine will use Docker Compose rather than running the binary directly.
- The custom-provider `config.yaml` from the current workstation will be copied securely or recreated there before login.

## Checklist

- [ ] 1) Prepare the home-machine runtime files
  - Target: home-machine copy of `config.yaml`, optional `docker-compose.local.yml`, `auths/`, `logs/`
  - Change: securely copy the current local runtime files or recreate them from your secret source; ensure `auths/` and `logs/` directories exist
  - Verify: `docker compose -f docker-compose.yml -f docker-compose.local.yml config` succeeds on the home machine

- [ ] 2) Start the proxy container on the home machine
  - Target: Docker runtime
  - Change: start the service with the same compose command used locally
  - Verify: `docker compose -f docker-compose.yml -f docker-compose.local.yml up -d` and `docker compose -f docker-compose.yml -f docker-compose.local.yml ps`

- [ ] 3) Run official Codex Team device login inside the container
  - Target: running `cli-proxy-api` container
  - Change: execute:
    - `docker exec -it cli-proxy-api ./CLIProxyAPI --codex-device-login --config /CLIProxyAPI/config.yaml`
  - Verify: the CLI prints a device code / verification URL flow and completes without error after browser authorization

- [ ] 4) Confirm auth persistence
  - Target: home-machine `auths/`
  - Change: verify that new Codex auth files were written through the mounted auth directory
  - Verify: inspect the host directory mounted to `/root/.cli-proxy-api` in `docker-compose.yml`

- [ ] 5) Confirm official Codex Team models are available
  - Target: running proxy
  - Change: query the model list using the local proxy API key
  - Verify:
    - `curl -H 'Authorization: Bearer <your-local-proxy-key>' http://127.0.0.1:8317/v1/models`
    - Expectation: official Codex Team models appear as unprefixed Codex/OpenAI models, while the custom provider remains under `linuxdo/*`

- [ ] 6) Add more official Codex Team accounts if needed
  - Target: same running container
  - Change: repeat the device-login command once per additional official account
  - Verify: additional auth files appear and the proxy logs show more auth entries loaded

## Notes

- Repo-supported official Codex flags:
  - `--codex-login`
  - `--codex-device-login`
- For a Dockerized setup, `--codex-device-login` is the safer default because it does not depend on callback port reachability.
- Do not add official Codex Team credentials under `codex-api-key:` in `config.yaml`; that section is for key-based custom upstreams such as the existing `linuxdo/*` provider.
- If the home machine does not need the port-collision workaround, you can omit `docker-compose.local.yml`; otherwise keep using the override that only publishes `8317`.
