# Rollback Notes: Set Up GHCR GitHub Actions Deployment To HK VPS

- Date: 2026-03-26
- Related: `spec.md`, `tasks.md`

## Rollback strategy

- Revert the workflow and `deploy/` changes from git if the rollout design needs to be abandoned.
- On the VPS, redeploy a previously known-good image by running the deploy workflow manually with an older GHCR image tag, or by SSHing to the server and re-running:

```bash
cd /opt/cliproxyapi
CLI_PROXY_IMAGE=ghcr.io/dovelanlan/cliproxyapi:<older-tag> bash scripts/remote-deploy.sh
```

- If Nginx or TLS setup causes issues, stop the public stack with:

```bash
cd /opt/cliproxyapi
docker compose -f compose.production.yml down
```

## Data / migration considerations

- No database or schema migrations are involved.
- `config.yaml`, `auths/`, `logs/`, and certificate files remain on the VPS and should not be deleted during rollback.
- GHCR package visibility changes may need to be reversed manually in GitHub Packages settings if you decide to make the package private again.

## Operational notes

- Monitoring/alerts to watch: container restart loops, Nginx TLS errors, GHCR pull failures, and GitHub deployment workflow failures.
- Known residual effects: reverting repo changes does not automatically remove files already copied to `/opt/cliproxyapi`; clean them manually if you are decommissioning the deployment layout.
