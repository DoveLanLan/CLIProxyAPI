# Rollback notes: native xAI Resin routing

## Immediate operational rollback

1. Set `xai-resin-proxy.enabled: false` in CPA `config.yaml`.
2. Set `ENABLE_XAI_RESIN_PROXY=false` in `/opt/cliproxyapi/.env`.
3. Reapply the CPA Compose stack with `bash scripts/remote-deploy.sh`.

Disabling the config hot-reloads request routing. Reapplying Compose also
removes the now-unused secret mounts. With both automatic backends disabled,
xAI returns to explicit per-auth proxy, CPA global proxy, or direct behavior in
the existing priority order.

To return to EgressProxyPool, first disable Resin on both config and deployment
sides, then enable `xai-proxy-pool` and its overlay. Never enable both automatic
backends in the same rollout.

The exact pre-rollout production files and previous image metadata are stored at
`/opt/cliproxyapi/backups/xai-resin-20260727T223545Z` on `bytevirt`. Restore the
saved `.env`, `config.yaml`, and deployment files from that directory before
reapplying the stack when an exact rollback is required.

For the same-auth retry follow-up, the immediate rollback directory is
`/opt/cliproxyapi/backups/xai-resin-retry-20260728T121546Z`. Restore its
`env.rollback` as `/opt/cliproxyapi/.env` and reapply the existing three-file
Compose stack to return from `cliproxyapi:xai-resin-retry-20260728` to
`cliproxyapi:xai-resin-20260728` without changing auth, Resin leases, or secret
files.

## State and secrets

- CPA auth files require no rollback because derived proxy URLs were never
  persisted.
- Resin Account leases can expire under Resin's normal lease policy; CPA does
  not delete them during rollback.
- Keep the identity key if Resin may be re-enabled. Reusing it preserves the
  same derived Accounts; deleting or rotating it changes every Account.
- The proxy token and identity key can be removed after the overlay is detached
  and rollback is confirmed. Secret deletion is an operator action and was not
  performed by this change.

## Source rollback

Revert the native Resin config, helper, executor wiring, tests, overlay, and
documentation together. No database, auth-file, or config-file migration needs
to be reversed.

The follow-up retry can also be reverted independently by restoring the prior
Resin branches in `xai_proxy_pool_executor.go` and `xai_websockets_executor.go`
together with their regression tests. It has no configuration or data migration.
