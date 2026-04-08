# Split Proxy Override

This optional override adds a local Squid proxy in front of your upstream HTTP proxy.
It lets CLIProxyAPI keep one global `proxy-url` while still sending local targets directly.

## When to use it

Use this when:

- your global upstream proxy rejects `localhost`, `127.0.0.1`, `172.x`, `192.168.x`, or Docker-only hostnames
- you still want the rest of CLIProxyAPI traffic to continue through that upstream proxy

## Files

- `docker-compose.split-proxy.yml`: compose override that adds the local Squid container
- `deploy/split-proxy/start.sh`: renders a minimal Squid config and starts the proxy

## Required config

Set the following environment variables before `docker compose up`:

```bash
export UPSTREAM_PROXY_HOST=proxy.example.com
export UPSTREAM_PROXY_PORT=3128
export UPSTREAM_PROXY_LOGIN='your-user:your-password'
```

Then point CLIProxyAPI's global proxy at the local split proxy:

```yaml
proxy-url: "http://split-proxy:3128"
```

Do not keep a local upstream on `http://localhost:8990` once the split proxy is enabled.
From Squid's point of view, `localhost` means the `split-proxy` container itself.

## Recommended upstream URLs

For local Claude-compatible services, prefer one of:

- `http://host.docker.internal:8990` when the upstream is published on the Docker host
- `http://kirors-kiro:8990` when the upstream container shares a Docker network with `split-proxy`

## Start

```bash
docker compose -f docker-compose.yml -f docker-compose.split-proxy.yml up -d
```

## Optional bypass tuning

Default direct-bypass hostnames:

- `localhost`
- `host.docker.internal`
- `kirors-kiro`

Default direct-bypass CIDRs:

- `127.0.0.0/8`
- `10.0.0.0/8`
- `172.16.0.0/12`
- `192.168.0.0/16`
- `169.254.0.0/16`
- `100.64.0.0/10`
- `::1/128`
- `fc00::/7`
- `fe80::/10`

Override them if needed:

```bash
export DIRECT_DOMAINS='localhost host.docker.internal kirors-kiro some.internal.domain'
export DIRECT_CIDRS='127.0.0.0/8 10.0.0.0/8 172.16.0.0/12 192.168.0.0/16'
```

## Notes

- This override assumes your upstream proxy is an HTTP proxy. If your upstream is HTTPS or SOCKS5, adjust the Squid peer config first.
- If `kirors-kiro` lives in a different compose project, `split-proxy` must be attached to a network where that name resolves. Otherwise use `host.docker.internal:8990`.
- `UPSTREAM_PROXY_LOGIN` uses Squid's `login=user:password` format. Escape literal `%` as `%%` and spaces as `%20`.
