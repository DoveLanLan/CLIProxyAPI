# Rollback notes: standalone EgressProxyPool

## Immediate operational rollback

1. Set CLIProxyAPI `xai-proxy-pool.enabled: false`.
2. Set `ENABLE_XAI_PROXY_POOL=false` in the CLIProxyAPI deployment environment.
3. Reapply the CLIProxyAPI Compose stack.

This restores the legacy explicit-auth-proxy, global-proxy, and direct routing
precedence. It does not delete any standalone pool data.

## Source rollback

- Revert the CLIProxyAPI extraction change to restore the previous embedded
  pool implementation and config shape.
- Restore the previous Mihomo overlay and private files before re-enabling it.
- The old CLIProxyAPI pool state was not deleted by this change.

## Standalone data

- `/opt/egress-proxy-pool/data/pool/state.json` contains non-secret lane and
  quarantine state.
- `data/pool/subscriptions.json` and `data/mihomo/config.yaml` contain write-only
  subscription URLs and must remain protected as secrets.
- Stopping controller alone leaves Mihomo data connections running, but new
  route/probe operations fail closed. Restarting Mihomo interrupts active proxy
  connections and should be done only during an approved maintenance window.
