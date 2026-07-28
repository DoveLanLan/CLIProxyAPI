# Rollback notes: bounded xAI Resin 402 retry

## Behavior-only rollback

Set `xai-resin-proxy.max-402-retries: 0` in CPA configuration and set
`XAI_RESIN_MAX_402_RETRIES=0` in the deployment environment. CPA will stop
calling the Resin admin API and return to the previous single-attempt sticky
Resin behavior. The admin URL and token file settings may remain unused.

## Full Resin rollback

1. Set `xai-resin-proxy.enabled: false`.
2. Set `ENABLE_XAI_RESIN_PROXY=false`.
3. Reapply the CPA deployment only after explicit production authorization.

This removes automatic Resin routing and its secret mounts. Existing explicit
per-auth proxies and CPA's prior fallback order remain available. Never enable
Resin and EgressProxyPool automatic routing together.

## State and source rollback

- No database or auth-file migration needs reversal.
- Derived Accounts and in-memory lease generations are not persisted by CPA.
- Deleting a Resin lease is safe state cleanup; Resin recreates it when the same
  Account next connects.
- Keep the identity key if Resin may be re-enabled so Account identities remain
  stable.
- Revert config, auth status error, Resin admin helper, executor retry wiring,
  deployment assets, tests, and docs together.

No production rollback point was created because this task made no production
changes.
