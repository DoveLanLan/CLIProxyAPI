# Production Deployment

This directory contains the production deployment artifacts for the forked VPS deployment at `api.heweili.top`.

## Topology

- Public traffic enters through Cloudflare.
- Cloudflare connects to Nginx on the VPS.
- Nginx proxies to the `cli-proxy-api` container on the internal Docker network.
- The management UI and `/v0/management` stay off the public domain.
- Management access is exposed separately on the Tailscale IP and a dedicated host port.

## Server Layout

Recommended root on the VPS:

```text
/opt/cliproxyapi/
  .env
  compose.production.yml
  nginx/conf.d/api.heweili.top.conf
  certs/origin.crt
  certs/origin.key
  data/config.yaml
  data/auths/
  data/logs/
  scripts/remote-deploy.sh
```

The deployment workflow will create the directory tree, but it will not create live runtime secrets for you.
The default production setup binds Nginx only to the VPS public IPv4 so it can coexist with Tailscale Serve/Funnel listeners on the Tailscale IP.

## One-Time Server Bootstrap

1. Copy this `deploy/` directory to `/opt/cliproxyapi/`.
2. Copy `.env.example` to `.env` if you want a local default image tag.
   Set:
   - `PUBLIC_BIND_IP` to the VPS public IPv4 that Cloudflare reaches
   - `TAILSCALE_BIND_IP` to the VPS Tailscale IPv4
   - `TAILSCALE_MANAGEMENT_PORT` to the private management port you want to use
3. Create `data/config.yaml` on the VPS.
4. Create `data/auths/` and place any existing auth files there if needed.
5. Ensure `data/logs/` exists and is writable.
6. Place the Cloudflare Origin CA certificate at `certs/origin.crt`.
7. Place the Cloudflare Origin CA private key at `certs/origin.key`.

## Cloudflare Steps

1. Create an `A` record for `api.heweili.top` pointing to `23.175.201.12`.
2. Keep the DNS record proxied (orange cloud enabled).
3. In `SSL/TLS`, set mode to `Full (strict)`.
4. In `SSL/TLS -> Origin Server`, generate an Origin CA certificate for `api.heweili.top`.
5. Save the generated certificate and private key onto the VPS under `certs/`.
6. Optionally enable `Always Use HTTPS` in Cloudflare.

## Tailscale Management Access

The production stack also binds the app container directly to the Tailscale IP on a dedicated management port. This keeps the public domain locked down while still allowing private admin access over the tailnet.

Required app config for management access:

```yaml
remote-management:
  allow-remote: true
  secret-key: "set-a-strong-management-secret"
  disable-control-panel: false
```

Recommended access URLs from a device already connected to the same tailnet:

- `http://100.67.99.9:18317/management.html#/`
- `http://t7y08hlk8c.tail3ae13e.ts.net:18317/management.html#/`

Notes:

- Keep the public Nginx rule blocking `/management.html` and `/v0/management`.
- The Tailscale access path is plain HTTP on the private tailnet, not HTTPS through Cloudflare.
- The management API still requires the configured management secret.

## GHCR Notes

- The image is intended to publish to `ghcr.io/dovelanlan/cliproxyapi`.
- After the first image publish, confirm the package visibility is `public` if you want the VPS to pull it without registry credentials.
- If you intentionally keep the package private, you must add a GHCR read token on the server and log in before pulling.

## GitHub Environment Secrets

Recommended production environment secrets:

- `PRODUCTION_SSH_PRIVATE_KEY`: dedicated deployment private key
- `PRODUCTION_SSH_KNOWN_HOSTS`: output of `ssh-keyscan -H 23.175.201.12`

No GHCR secret is required for image publication from Actions because the workflow can publish with `GITHUB_TOKEN` when it runs in this repository.

## Management Access

Public management is intentionally disabled.

Use one of these private-access methods instead:

- Tailscale: `http://100.67.99.9:18317/management.html#/`
- MagicDNS/Tailnet hostname: `http://t7y08hlk8c.tail3ae13e.ts.net:18317/management.html#/`
- SSH tunnel fallback: `ssh -L 8317:127.0.0.1:8317 root@23.175.201.12`

This keeps `/management.html` and `/v0/management` off the public internet while still allowing operator access.
