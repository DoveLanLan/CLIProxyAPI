# Spec: Integrate CPA-Manager Panel and Usage Monitoring

- Date: 2026-05-14
- Owner(s): hewei
- Related: proposal.md, tasks.md

## Repo Snapshot

- Go proxy/server repository with management APIs under `internal/api/handlers/management`, runtime usage publication through `sdk/cliproxy/usage`, and external management panel handling under `internal/managementasset`.
- No first-party browser frontend is maintained in-tree; web dashboard changes should use the management asset integration and Management API contracts.
- Required gate after source changes: `go build -o test-output ./cmd/server && rm test-output`; broader regression gate is `go test ./...`.

## Scope

### In scope

- Default CPA panel source changes to `https://github.com/seakee/CPA-Manager`.
- Add in-memory usage queue controlled by management availability and `usage-statistics-enabled`.
- Add `GET /v0/management/usage-queue?count=N` and `GET /v0/management/api-key-usage`.
- Add deployment examples for the external CPA-Manager Usage Service.

### Out of scope

- Bundling CPA-Manager source code.
- Full upstream version upgrade.
- RESP queue protocol compatibility.
- Any `internal/translator/**` changes.

## Acceptance Criteria (testable)

1. `/management.html` downloads CPA-Manager `management.html` from GitHub Releases by default. (Verify: config default/unit review/manual first load)
2. `usage-statistics-enabled: true` causes usage records to be queued and `false` stops new queue publication. (Verify: unit tests)
3. `GET /v0/management/usage-queue?count=N` returns up to `N` oldest JSON records and pops them. (Verify: management handler tests)
4. `GET /v0/management/api-key-usage` groups API-key auths by provider and `base_url|api_key`. (Verify: management handler tests)
5. Compose/deploy examples include one persistent `seakee/cpa-manager` service with CPA upstream and management key configuration. (Verify: file review)

## Behavior / Requirements

- Queue publication is disabled when management routes are disabled and cleared when disabled.
- Queue retention defaults to 60 seconds and clamps to 3600 seconds.
- `usage-queue` requires existing management authentication.
- Usage queue records are JSON objects when source data is valid; non-object/invalid queue items are not produced by the plugin.
- CPA-Manager Usage Service remains external and stores its SQLite data under `/data`.

## Edge Cases

- `count` missing defaults to one record; non-positive or non-integer `count` returns 400 without popping.
- Empty queue returns an empty JSON array.
- Missing management secret keeps management routes and queue disabled.
- If a local override for `panel-github-repository` is set, it continues to override the default.

## Compatibility Notes

- Backwards compatibility: existing management panel config overrides remain supported.
- Data/migrations: no database migration; queue is in-memory and best-effort.
- Config/flags: add `redis-usage-queue-retention-seconds` to YAML config.
- Security: no secrets are logged; deployment docs keep management off public ingress.

## API/UX Decisions

- CPA-hosted panel path remains `/management.html`.
- Request monitoring requires separate Usage Service and `usage-statistics-enabled: true`.
- Production docs continue recommending Tailscale/private access for management.
