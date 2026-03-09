# Logging Guidelines

> How logging is implemented and what should be logged in CLIProxyAPI.

---

## Overview

The repo uses `logrus` with a custom formatter and optional file output. There are two main logging streams:

- application/runtime logs through the global logger
- per-request request/response logs through the request logger

Primary evidence: `internal/logging/global_logger.go`, `internal/logging/request_logger.go`, `internal/api/middleware/request_logging.go`

---

## Log Levels

- `debug`: internal state transitions, optional diagnostics, watcher churn, updater throttling
- `info`: lifecycle events such as startup, route registration, connected runtime providers
- `warn`: recoverable failures and degraded fallback behavior
- `error`: startup blockers, persistence failures, or request-processing failures that need attention

Use the existing severity choices in adjacent code instead of inventing a new local convention per package.

---

## Structured Logging

- `logging.SetupBaseLogger()` configures the shared logger once.
- The custom formatter prints timestamp, request ID, level, call site, message, and selected fields.
- Request-scoped logs should carry the generated request ID when possible.
- Gin’s default writers are redirected through the same logger so framework output stays consistent.

Primary evidence: `internal/logging/global_logger.go`, `internal/logging/requestid.go`

---

## What to Log

- startup mode changes and important configuration pivots
- management route registration and availability changes
- storage bootstrap or sync failures
- websocket provider connect/disconnect events
- request/response logs for non-management routes when enabled
- request-log write failures or decompression problems as warnings

---

## What NOT to Log

- full management secrets
- raw OAuth tokens, cookies, API keys, or auth file contents
- management endpoint bodies that would expose secrets
- large multipart payloads or unbounded bodies when request logging is in error-only mode

The request logger intentionally skips management routes and limits body capture in low-overhead mode. Preserve those protections.

Primary evidence: `internal/api/middleware/request_logging.go`, `internal/logging/request_logger.go`
