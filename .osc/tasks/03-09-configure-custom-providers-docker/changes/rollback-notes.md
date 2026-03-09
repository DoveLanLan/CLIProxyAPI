# Rollback Notes: Configure Custom Providers And Docker Compose Startup

- Date: 2026-03-09
- Related: `change-summary.md`, `spec.md`

## Rollback strategy

1. Stop and remove the local stack:
   - `docker compose -f docker-compose.yml -f docker-compose.local.yml down --remove-orphans`
2. Remove the local runtime files if you no longer want this setup:
   - `config.yaml`
   - `docker-compose.local.yml`
3. Remove local auth state only if you want to discard future OAuth/device-login accounts as well:
   - `auths/`

## Data / migration considerations

- No schema or database migration was introduced.
- Runtime credentials were stored only in the local gitignored `config.yaml`.
- Future official Codex Team accounts will be stored under the mounted `auths/` directory; those are independent of the custom API-key provider config.

## Operational notes

- Monitoring/alerts to watch: container restart loops, failed upstream connectivity, and requests accidentally sent to the wrong prefixed provider.
- Known residual effects: `docker-compose.local.yml` is a local override file and should be treated as workstation-local runtime config unless you intentionally want to share it.
