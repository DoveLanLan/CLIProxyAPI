# Spec: xAI streamed free-usage failover and Grok account-pool repair

- Date: 2026-07-23
- Owner(s): hewei
- Related: `proposal.md`, `tasks.md`

## Repo Snapshot (from step 0)

- Server entrypoint: `cmd/server`; provider executors: `internal/runtime/executor`.
- Shared auth scheduling and cooldowns: `sdk/cliproxy/auth`.
- Public protocol handling: `sdk/api/handlers`; protected translators: `internal/translator`.
- Config/runtime reload: `internal/config`, `sdk/cliproxy`, `internal/watcher`.
- Production deployment: Docker image plus `deploy/` and GitHub Actions.
- Toolchain: Go 1.26, `gofmt`, package tests, server build.
- CI requires `go build -o test-output ./cmd/server` and guards translator paths.
- OSC requires proposal/spec/tasks before source edits and a persisted gate report.
- Confidence: High.
- Evidence: `AGENTS.md`, `go.mod`, `config.example.yaml`, `.github/workflows/`, `.osc/spec/project-spec.md`.

## Scope

### In scope

- xAI HTTP SSE error classification before translation.
- Free-usage exhaustion mapped to HTTP 429 plus a 24-hour retry hint.
- Auth-manager credential cooldown and failover regression coverage.
- Production cooldown persistence, routing, and Grok inspection policy.
- Safe disabled-account reinspection, recovery, backup, and hard-dead deletion.

### Out of scope

- Translator changes.
- Generic provider retry redesign.
- Paid xAI subscriptions.
- Deletion of quota-exhausted, healthy, ambiguous, or transient probe-error accounts.

## Acceptance Criteria (testable)

1. A streamed payload with `subscription:free-usage-exhausted` becomes a status-429 stream error carrying a 24-hour retry hint before downstream data. (Verify: focused executor test.)
2. When the first xAI credential returns the streamed quota error, the auth manager cools it and succeeds with a second credential without exposing the first error downstream. (Verify: auth-manager stream failover test.)
3. Normal xAI SSE response events continue through the existing translation path. (Verify: negative/normal stream test.)
4. No files under `internal/translator/**` change. (Verify: `git diff --name-only`.)
5. Production persists cooldowns and uses round-robin selection after deployment. (Verify: config grep, `.cds` creation after a controlled failure, management status.)
6. Permanent inspection classes exclude rolling quota and transient probe errors. (Verify: deployed script configuration.)
7. Disabled credentials are freshly classified before action; only hard-dead classes are deleted, and exact deletion targets have a restricted backup. (Verify: aggregate inspection report, backup manifest/count, post-delete management aggregate.)
8. Focused tests, formatting, full tests, and the required server build pass. (Verify: recorded quality-gate output.)

## Behavior / Requirements

The HTTP SSE executor must inspect each decoded xAI data object before sending it
to protocol translation. Explicit free-usage exhaustion is emitted as
`StreamChunk.Err` using the existing `xaiStatusErr` behavior so the shared auth
manager records a model cooldown and selects another credential. Successfully
translated events remain byte-compatible with existing behavior.

Production inspection must distinguish temporary and permanent states:

- Recoverable: `healthy`, `quota_exhausted`, rate limits, network/probe errors, and unknown/other results.
- Hard-dead deletion candidates: `invalid_grant`, `reauth` after a fresh failed credential probe, `deactivated`, `banned`, and explicit permanent permission denial.
- A credential is never deleted solely because it is disabled or quota-exhausted.
- Healthy disabled credentials may be re-enabled only after a fresh probe.

## Edge Cases

- Error payloads may use a top-level string `error`, nested `error.code`, or an explicit status field.
- Generic 429s without the free-usage marker retain existing backoff semantics.
- An error after meaningful payload bytes cannot be transparently retried; it must still mark the credential failed.
- Empty/malformed SSE data must retain existing translator/error behavior.
- Interrupted inspection leaves credentials unchanged until a complete result is available.
- Backup or delete failure stops the batch and preserves the remaining targets.

## Compatibility Notes

- Backwards compatibility: Normal xAI response translation is unchanged; only explicit upstream error objects change control flow.
- Data/migrations: No repository schema migration. Production creates cooldown `.cds` files and an operational backup/manifest.
- Config/flags: `save-cooldown-status: true`; routing changes to `round-robin`.

## API/UX Decisions (if applicable)

- Inputs/outputs: Public API shape is unchanged when failover succeeds.
- States/errors: If all eligible credentials fail, the existing 429/model-cooldown response remains authoritative.
- Telemetry/logging: Log aggregate outcomes only; never log auth tokens, keys, or credential contents.
- Accessibility/i18n: Not applicable.
