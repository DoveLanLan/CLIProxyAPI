# Production Deployment

This directory contains the production deployment artifacts for the forked VPS deployment at `api.heweili.top`.

Additional guide:

- Chinese split-proxy setup: [SPLIT_PROXY_SETUP_CN.md](/root/Projects/Go/src/CLIProxyAPI/deploy/SPLIT_PROXY_SETUP_CN.md)

## Topology

- Public traffic enters through Cloudflare.
- Cloudflare connects to the shared `vps-gateway-nginx` container on the VPS.
- The shared gateway proxies `api.heweili.top` to the `cli-proxy-api` container on the `vps-gateway` Docker network.
- The management UI and `/v0/management` stay off the public domain.
- Management access is exposed separately on the Tailscale IP and a dedicated host port.
- CPA-Manager Usage Service is exposed separately on the Tailscale IP for persistent request monitoring.

## Server Layout

Recommended root on the VPS:

```text
/opt/cliproxyapi/
  .env
  compose.production.yml
  compose.production.split-proxy.yml
  data/config.yaml
  data/auths/
  data/logs/
  data/cpa-manager/
  data/logs/split-proxy/
  split-proxy/start.sh
  scripts/remote-deploy.sh
```

The deployment workflow will create the directory tree, but it will not create live runtime secrets for you.
The shared gateway stack lives at `/opt/vps-gateway` and is the only container that should bind the VPS public `80/443` ports.

## One-Time Server Bootstrap

1. Copy this `deploy/` directory to `/opt/cliproxyapi/`.
2. Copy `.env.example` to `.env` if you want a local default image tag.
   Set:
   - `GATEWAY_NETWORK` to the shared Docker network, usually `vps-gateway`
   - `LOCAL_CLAUDE_NETWORK` to the Docker network shared with local Claude-compatible services, usually `cli-proxy-api-proxy`
   - `GATEWAY_ROOT` to the gateway stack root, usually `/opt/vps-gateway`
   - `GATEWAY_CONTAINER` to the gateway nginx container, usually `vps-gateway-nginx`
   - `TAILSCALE_BIND_IP` to the VPS Tailscale IPv4
   - `TAILSCALE_MANAGEMENT_PORT` to the private management port you want to use
   - `TAILSCALE_CPA_MANAGER_PORT` to the private CPA-Manager Usage Service port
   - `CPA_MANAGEMENT_KEY` to the plaintext Management Key used by CPA and CPA-Manager
   - `ENABLE_SPLIT_PROXY=true` only if you want the local split-proxy sidecar
   - `UPSTREAM_PROXY_HOST` / `UPSTREAM_PROXY_PORT` / `UPSTREAM_PROXY_LOGIN` only when split-proxy is enabled
3. Create `data/config.yaml` on the VPS.
4. Create `data/auths/` and place any existing auth files there if needed.
5. Ensure `data/logs/` exists and is writable.
   Ensure `data/cpa-manager/` exists and is writable for CPA-Manager SQLite data.
   When split-proxy is enabled, `data/logs/split-proxy/` will be used for Squid logs.
6. Ensure the shared gateway stack exists and mounts the certificate directory expected by `api.heweili.top.conf`.

## Split Proxy Option

If your global upstream proxy rejects `localhost`, Docker hostnames, or private subnets, keep those runtime secrets on the server in `/opt/cliproxyapi/.env` instead of GitHub Actions secrets.

Example:

```env
ENABLE_SPLIT_PROXY=true
LOCAL_CLAUDE_NETWORK=cli-proxy-api-proxy
UPSTREAM_PROXY_HOST=proxy.example.com
UPSTREAM_PROXY_PORT=3128
UPSTREAM_PROXY_LOGIN=your-user:your-password
DIRECT_DOMAINS="localhost host.docker.internal kiro-rs kirors-kiro"
```

Then set the app config to use the local sidecar:

```yaml
proxy-url: "http://split-proxy:3128"
```

For a local Claude-compatible upstream, do not keep `http://localhost:8990` once split-proxy is enabled.
Use `http://host.docker.internal:8990` or a shared-network service name instead.
If you use a Docker service name such as `http://kiro-rs:8990`, make sure `LOCAL_CLAUDE_NETWORK` points to the existing Docker network that contains that service. The production default is `cli-proxy-api-proxy`, and the deploy script fails early if that network is missing.

## Cloudflare Steps

1. Create an `A` record for `api.heweili.top` pointing to `23.175.201.12`.
2. Keep the DNS record proxied (orange cloud enabled).
3. In `SSL/TLS`, set mode to `Full (strict)`.
4. In `SSL/TLS -> Origin Server`, generate an Origin CA certificate for `api.heweili.top`.
5. Save the generated certificate and private key in the certificate directory mounted by the shared gateway as `/etc/nginx/certs/`.
6. Optionally enable `Always Use HTTPS` in Cloudflare.

## Tailscale Management Access

The production stack also binds the app container directly to the Tailscale IP on a dedicated management port. This keeps the public domain locked down while still allowing private admin access over the tailnet.

Required app config for management access:

```yaml
remote-management:
  allow-remote: true
  secret-key: "set-a-strong-management-secret"
  disable-control-panel: false
  panel-github-repository: "https://github.com/seakee/CPA-Manager"

usage-statistics-enabled: true
redis-usage-queue-retention-seconds: 60
```

Recommended access URLs from a device already connected to the same tailnet:

- `http://100.67.99.9:18317/management.html#/`
- `http://t7y08hlk8c.tail3ae13e.ts.net:18317/management.html#/`

CPA-Manager Usage Service is available on a separate private port:

- `http://100.67.99.9:18318/management.html#/`
- `http://t7y08hlk8c.tail3ae13e.ts.net:18318/management.html#/`

Notes:

- Keep the public Nginx rule blocking `/management.html` and `/v0/management`.
- The Tailscale access path is plain HTTP on the private tailnet, not HTTPS through Cloudflare.
- The management API still requires the configured management secret.
- Run only one CPA-Manager Usage Service per CPA instance because it drains `/v0/management/usage-queue`.

## GHCR Notes

- The image is intended to publish to `ghcr.io/dovelanlan/cliproxyapi`.
- After the first image publish, confirm the package visibility is `public` if you want the VPS to pull it without registry credentials.
- If you intentionally keep the package private, you must add a GHCR read token on the server and log in before pulling.

## GitHub Environment Secrets

Recommended production environment secrets:

- `PRODUCTION_SSH_PRIVATE_KEY`: dedicated deployment private key
- `PRODUCTION_SSH_KNOWN_HOSTS`: output of `ssh-keyscan -H 23.175.201.12`

No GHCR secret is required for image publication from Actions because the workflow can publish with `GITHUB_TOKEN` when it runs in this repository.

## Shared Gateway Notes

`CLIProxyAPI` no longer owns the public nginx container. Its deployment script installs `nginx/conf.d/api.heweili.top.conf` into `${GATEWAY_ROOT:-/opt/vps-gateway}/nginx/conf.d/` and reloads `${GATEWAY_CONTAINER:-vps-gateway-nginx}` after `nginx -t` passes.

The shared gateway should be the only container binding public `80/443`. App stacks join `${GATEWAY_NETWORK:-vps-gateway}` and are reached by Docker service name.

## Management Access

Public management is intentionally disabled.

Use one of these private-access methods instead:

- Tailscale: `http://100.67.99.9:18317/management.html#/`
- MagicDNS/Tailnet hostname: `http://t7y08hlk8c.tail3ae13e.ts.net:18317/management.html#/`
- SSH tunnel fallback: `ssh -L 8317:127.0.0.1:8317 root@23.175.201.12`

For persistent request monitoring, open CPA-Manager Usage Service on the private port configured by
`TAILSCALE_CPA_MANAGER_PORT` and configure the CPA-hosted panel to use that Usage Service URL.

This keeps `/management.html` and `/v0/management` off the public internet while still allowing operator access.
