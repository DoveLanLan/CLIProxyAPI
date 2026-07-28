# Tasks: native Resin routing for xAI

- [x] Add and normalize `XAIResinProxyConfig`.
- [x] Add fail-closed Resin router with secret-file loading and HMAC identities.
- [x] Wire Resin routing into `XAIAutoExecutor` before EgressProxyPool routing.
- [x] Preserve explicit proxy precedence and refresh restoration.
- [x] Add config parsing, deterministic identity, invalid config, HTTP proxy-auth,
      WebSocket target, and precedence tests.
- [x] Add `config.example.yaml` and Chinese deployment documentation.
- [x] Update project spec and config diff visibility.
- [x] Run `gofmt -w .`.
- [x] Run focused tests and relevant race tests.
- [x] Run `go test ./...`.
- [x] Run `go build -o test-output ./cmd/server && rm test-output`.
- [x] Write change summary, regression checklist, rollback notes, and quality gate.
- [x] Audit the live `bytevirt` CPA/Resin deployment and identify missing pieces.
- [x] Back up and install the Resin overlay, deployment flags, config block, and secrets.
- [x] Deploy a CPA image containing the reviewed Resin integration.
- [x] Verify the real CPA -> Resin -> xAI network path and classify the upstream 402 response.
- [x] Record the live rollout result and final production rollback point.
- [x] Add one safe same-auth retry for pre-response Resin network failures.
- [x] Cover non-streaming success, repeated failure, stream bootstrap replay,
      mid-stream no-replay, cancellation, and no-Egress-fallback behavior.
- [x] Re-run formatting, focused/full tests, race tests, vet, and server build.
- [x] Redeploy CPA and verify a forced first Resin failure is hidden by the
      same-Account retry in production.
