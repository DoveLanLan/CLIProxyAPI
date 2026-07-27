# Regression Checklist: xAI dedicated rotating proxy pool

- [x] Missing/disabled pool preserves legacy xAI routing.
- [x] Explicit auth `proxy_url` bypasses the pool.
- [x] Enrolled auth receives a stable lane proxy and does not chain through the
  global proxy.
- [x] Rendezvous hashing is deterministic and minimizes movement when lanes are
  added.
- [x] Rollout assignment is deterministic.
- [x] Default lane rate is 30 starts/minute with burst 3 and bounded queue 30.
- [x] Queue overflow is a request-scoped 503.
- [x] Pool-local 503 responses include CPA-managed `Retry-After` without
  enabling arbitrary upstream-header passthrough.
- [x] Only exact spending-limit 402 triggers one same-auth A/B attempt.
- [x] Alternate success quarantines the old public IP and promotes the verified
  node.
- [x] Repeated exact 402 remains a credential failure.
- [x] Ambiguous/unavailable A/B returns a request-scoped 503.
- [x] Pre-connect retry requires a negative Mihomo node-health result.
- [x] A second proxy network failure is request-scoped.
- [x] Mid-response failure is not replayed and the configured threshold rotates
  only future connections.
- [x] A lane node change changes xAI WebSocket session identity.
- [x] OAuth refresh restores the original auth proxy before returning the auth
  for persistence.
- [x] Startup initialization can recover through a later provider refresh.
- [x] State reload preserves selections, failure windows, and unexpired
  quarantines without persisting auth IDs or secrets.
- [x] Management status is redacted and control endpoints remain under existing
  Management API authentication.
- [x] xAI management inspection uses the registered xAI executor.
- [x] Compose overlay has no published host port, TUN, host network, privileged
  mode, or added capability.
- [x] Mihomo bootstrap is empty, fail closed, and contains no live provider URL.
- [x] API registry accepts multiple independent providers and enforces the
  configured provider-count limit.
- [x] Subscription URLs are absent from list, status, and error responses.
- [x] Disabled drafts persist without reload; enabled create/update requires a
  successful provider refresh and at least one live node.
- [x] Candidate reload, provider activation, generated-file persistence, and
  registry persistence failures restore the prior runtime and registry.
- [x] Failed enabled URL replacement preserves the previous URL and generated
  configuration; startup repairs generated-config drift from the registry.
- [x] Subscription mutations are serialized and stale revisions return 412 with
  the current `ETag`; malformed or missing revisions are rejected.
- [x] HTTPS/userinfo/fragment/localhost/private-literal validation, request body
  size, registry symlink, and conflicting storage paths are covered.
- [x] Disable drains active lanes; hard delete rejects enabled/in-use providers
  and removes the provider cache only after disable.
- [x] Manual provider check records only a timestamp and safe error code.
- [x] Registry and generated configuration are atomically written mode `0600`.
- [x] Shell syntax, Compose render, and Mihomo native config validation pass.
- [x] Pinned Mihomo accepts the generated configuration and its live local
  controller accepts `PUT /configs?force=true` payload reload.
- [x] Focused tests, full tests, required build, and focused race checks pass.
- [x] Repository-wide vet reports only the six documented pre-existing findings.
- [x] `internal/translator/**` remains unchanged.
- [x] No production SSH, installation, configuration mutation, or restart was
  performed.
