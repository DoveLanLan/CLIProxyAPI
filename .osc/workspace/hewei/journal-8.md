# hewei journal 8

- Date: 2026-05-14
- Title: Integrate CPA-Manager panel and usage monitoring
- Commit:

## Summary

Conclusions/decisions: CPA-Manager should replace the default `/management.html` panel via the existing release asset updater, while request monitoring stays in a separate `seakee/cpa-manager` Usage Service. Only the minimal HTTP usage queue compatibility surface was backported; no full upstream upgrade and no CPA-Manager source vendoring.

What changed: added `internal/redisqueue`, `GET /v0/management/usage-queue`, `GET /v0/management/api-key-usage`, auth recent request counters, usage record alias/auth metadata, CPA-Manager panel defaults, `redis-usage-queue-retention-seconds`, and compose/deploy examples for the Usage Service.

Verification: ran `gofmt`; targeted tests for redisqueue, management handlers, auth, usage packages; `go test ./...`; and `go build -o test-output ./cmd/server && rm test-output`. All passed.

Risks/rollback: Usage Service drains the queue, so only one consumer should run per CPA instance. Existing configs that explicitly set the old panel repo will keep using it until updated. Rollback is a straight revert plus stopping/removing the `cpa-manager` service and its `/data` volume if desired.
