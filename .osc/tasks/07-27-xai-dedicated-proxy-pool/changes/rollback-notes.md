# Rollback Notes: xAI dedicated rotating proxy pool

## Runtime rollback

1. Set `xai-proxy-pool.enabled: false` in the runtime config.
2. Confirm xAI requests have returned to the legacy precedence:
   explicit auth proxy, global proxy, then context transport/direct behavior.
3. Only after the pool is disabled, remove the optional
   `compose.production.xai-proxy.yml` overlay and stop the Mihomo sidecar.

Disabling the feature is the fastest rollback and does not require deleting any
auth file or credential state.

## Subscription-management rollback

1. Stop issuing subscription mutations and retain the latest registry revision.
2. Disable `xai-proxy-pool.enabled` first if traffic must stop using the current
   generated providers immediately.
3. Set `xai-proxy-pool.subscription-management.enabled: false` to remove the CRUD
   surface. This does not itself delete private registry or provider data.
4. If reverting to a static Mihomo configuration, restore the previous private
   config/mount layout and validate it before reloading Mihomo. That production
   operation requires separate authorization.

Each failed enabled mutation automatically reloads the previous payload and
restores the generated file when it had already changed. A hard provider delete
cannot be rolled back from API output because the URL is intentionally
write-only; retain an operator-managed encrypted backup if deletion recovery is
required.

## Repository rollback

- Revert the feature change set, including the new config schema, pool runtime,
  xAI executor integration, management endpoints, tests, and deployment assets.
- Keep or remove the ignored production files under `deploy/secrets/` according
  to operator policy; they are not repository content.
- The pool state file can be archived or deleted independently. It contains
  lane/node names, public egress IPs, counters, failure timestamps, and
  quarantine expiry only.
- Treat the subscription registry, generated Mihomo config, and provider cache
  as secrets. Preserve mode `0600` and do not copy their contents into tickets,
  logs, or Git history.

## Verification after rollback

- Run `go test ./...`.
- Run `go build -o test-output ./cmd/server && rm test-output`.
- Render the base production Compose file without the xAI overlay.
- Verify the xAI proxy-pool Management API reports disabled or is unavailable.

No rollback action was executed against production during this task.
