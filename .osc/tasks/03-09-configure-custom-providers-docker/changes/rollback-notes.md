# Rollback Notes: Configure Custom Providers And Docker Compose Startup

- Date: 2026-03-09
- Related: `change-summary.md`, `spec.md`

## Rollback strategy

1. Stop and remove the local stack:
   - `docker compose -f docker-compose.yml -f docker-compose.local.yml down --remove-orphans`
2. Revert the host-gateway mapping change in the tracked compose file:
   - remove `extra_hosts: ["host.docker.internal:host-gateway"]` from `docker-compose.yml`, or revert the commit that introduced it
3. Restore the previous local Claude upstream URL if you intentionally want the old behavior again:
   - change `config.yaml` `claude-local.base-url` back from `http://host.docker.internal:8990`
4. Remove the local runtime files if you no longer want this setup:
   - `config.yaml`
   - `docker-compose.local.yml`
5. Remove local auth state only if you want to discard future OAuth/device-login accounts as well:
   - `auths/`

## Data / migration considerations

- No schema or database migration was introduced.
- Runtime credentials were stored only in the local gitignored `config.yaml`.
- Future official Codex Team accounts will be stored under the mounted `auths/` directory; those are independent of the custom API-key provider config.

## Operational notes

- Monitoring/alerts to watch: container restart loops, failed upstream connectivity, and requests accidentally sent to the wrong prefixed provider.
- Known residual effects: `docker-compose.local.yml` is a local override file and should be treated as workstation-local runtime config unless you intentionally want to share it.
- If `host.docker.internal` stops resolving after a Docker Engine upgrade or environment change, re-check the compose `extra_hosts` mapping before debugging provider credentials.
