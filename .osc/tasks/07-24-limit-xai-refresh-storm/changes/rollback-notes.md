# Rollback Notes: Limit xAI Credential Refresh Storm

## Production Config

- Pre-change backup: `/opt/cliproxyapi/data/config.yaml.bak.20260724T153800Z`
- Restore it over `/opt/cliproxyapi/data/config.yaml` with mode 0600 and owner `root:root`, then replace the container.

## Production Image

- The previous configured image remains `ghcr.io/dovelanlan/cliproxyapi:main` in `/opt/cliproxyapi/.env`.
- Recreate only `cli-proxy-api` with the normal production compose command and `CLI_PROXY_IMAGE=ghcr.io/dovelanlan/cliproxyapi:main` to roll back the local image.

## Source

- Revert the five Go-file changes in this task to restore upstream refresh scheduling and error classification.

## Data

- No credential files were deleted by this change.
- Grok Inspection independently disabled 14 newly detected permanent failures before deployment; those management actions are not reverted by restoring code or config.
