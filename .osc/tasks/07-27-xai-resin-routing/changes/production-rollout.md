# Production rollout: native xAI Resin routing

## Target and rollback point

- Host alias: `bytevirt`
- CPA root: `/opt/cliproxyapi`
- Resin root: `/opt/resin`
- Pre-rollout backup: `/opt/cliproxyapi/backups/xai-resin-20260727T223545Z`
- CPA image: `cliproxyapi:xai-resin-20260728`
- CPA build version: `xai-resin-20260728`
- CPA build commit: `4388f6e4+xai-resin`
- Resin image: `ghcr.io/dovelanlan/resin:sha-74cc7879afe73d82d09bdfdbb75495ce24af77ab`

## Applied configuration

- CPA and Resin share the external `vps-gateway` Docker network.
- `xai-resin-proxy.enabled` and `ENABLE_XAI_RESIN_PROXY` are true.
- `xai-proxy-pool.enabled` and `ENABLE_XAI_PROXY_POOL` are false.
- CPA routes to `http://resin:2260` using Platform `Default`.
- The existing Resin V1 proxy token and a new CPA-only identity key are mounted
  from mode `0600` files into CPA as read-only secrets.
- The CPA deployment script supports the local rollout image without attempting
  a registry pull.

## Acceptance evidence

- CPA container: running, expected image and network, HTTP root 200.
- Resin container: running and healthy.
- Resin `Default` Platform: 422 routable nodes at final verification.
- Dynamic xAI leases: 89 matching `^xai-[0-9a-f]{32}$` and zero malformed
  `xai-` leases.
- Real CPA requests caused new Resin leases and reached xAI. The sampled xAI
  credentials returned `402 personal-team-blocked:spending-limit` after routing;
  this confirms the proxy path while identifying an independent upstream
  credential spending/subscription condition.
- No Resin configuration errors appeared in recent CPA logs.
- No manual per-credential Account provisioning or auth-file mutation was used.

## Cleanup

The temporary upload directory `/tmp/cliproxy-xai-resin-rollout` was removed
after acceptance. The production backup, secret files, and deployed local image
were retained.

## Exhaustive enabled-credential check

On 2026-07-28, all 1,291 enabled xAI credential files were tested exactly once
with `grok-3-mini` through their independently derived Resin Accounts. The
request used the same CLI chat-proxy URL, credential headers, Responses payload,
Resin V1 authentication, and HMAC Account derivation as CPA.

- 1,283 credentials reached xAI and returned
  `402 personal-team-blocked:spending-limit`.
- 8 credentials returned 401 with their current access tokens; a second check
  produced the same result.
- Three initial Resin CONNECT failures were retried and then returned the same
  xAI spending-limit 402.
- No enabled credential returned 200.
- All 1,291 expected enabled-credential Accounts were present in Resin after the
  check; the missing count was zero.
- Resin contained 2,411 valid `xai-<32 lowercase hex>` Accounts in total and no
  malformed `xai-` Account names.
- CPA still returned HTTP 200 and Resin remained healthy with 402 routable nodes
  after the check.

This separates the two outcomes: native CPA -> Resin routing is operating for
the complete enabled credential set, while the current xAI credentials do not
contain an account able to complete this model request successfully.

## Credential backup and polling impact

All 4,828 xAI credential files were copied into a root-only archive on
`bytevirt` without removing or modifying the active source files:

- Directory: `/var/backups/cliproxyapi/xai-auths-20260728T010747Z`
- Archive: `xai-auths.tar.gz`
- Archived file count: 4,828, matching the source count
- Archive SHA-256:
  `c6746fc3217fd3a84266d157f64052dbbe7e1fe2243aa73e14719a17aa686157`
- Archive ownership/mode: `root:root`, `0600`
- Gzip integrity and archive member-list comparison: pass

The VPS has an active `grok-inspection.timer` that runs every five minutes in
incremental mode. Moving xAI files out of CPA's auth directory would cause the
file watcher to remove those runtime auths and the built-in auto-refresh loop to
unschedule them. The systemd timer itself would remain active, but would have no
xAI candidates until a new credential is added. The backup operation did not
move the files, so current routing and polling remain unchanged.

## Credential evacuation

After operator confirmation, all 4,828 xAI files were evacuated from CPA's
active auth directory. CPA and the inspection timer were stopped during the
move so neither auto-refresh nor inspection could recreate or mutate a file
mid-migration.

- Active auth-directory xAI files after restart: 0
- Runtime xAI auth entries after restart: 0
- Evacuated xAI files: 4,828
- Unrelated auth files retained in the active directory: 17
- Individual-file quarantine:
  `/var/backups/cliproxyapi/xai-auths-20260728T010747Z/migrated-auths`
- Exact post-migration archive:
  `/var/backups/cliproxyapi/xai-auths-20260728T010747Z/xai-auths-migrated-current.tar.gz`
- Post-migration archive SHA-256:
  `46746ea451873c574f00274c2ba0284ea00526602a4070e3a75a3f57964abec3`
- Post-migration archive count and integrity check: 4,828, pass

CPA returned HTTP 200 after restart. `grok-inspection.timer` was restored to
active state, and a manual zero-runtime-xAI inspection completed successfully
with no disable or recovery action. A subsequent source/runtime check remained
at zero, so only credentials registered after this evacuation can enter the
active xAI pool.
