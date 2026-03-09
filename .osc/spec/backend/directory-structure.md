# Directory Structure

> How backend code is organized in CLIProxyAPI.

---

## Overview

The backend is organized around a small server entrypoint, a large `internal/` implementation tree, and a reusable `sdk/` surface. New code should go where its ownership and reuse level are already implied by the existing layout.

---

## Directory Layout

```text
cmd/
  server/                  runnable CLIProxyAPI server binary

internal/
  api/                     Gin server, middleware, route modules, management handlers
  auth/                    provider-specific auth helpers
  cmd/                     CLI login/import commands and startup helpers
  config/                  YAML config structs, normalization, validation
  logging/                 global logger, request logger, request IDs
  managementasset/         external management panel asset updater/serving helpers
  runtime/                 server runtime helpers and executors
  store/                   file/Postgres/Git/Object-backed state persistence
  tui/                     Bubble Tea terminal management UI
  usage/                   usage statistics and plugins
  watcher/                 config/auth file watching and delta emission
  wsrelay/                 websocket relay support

sdk/
  api/                     protocol handlers shared with the server
  auth/                    auth abstractions for embedders
  cliproxy/                embeddable service builder, runtime, auth core
  config/                  SDK-facing config types
  logging/                 SDK-facing logging helpers
  translator/              schema translation registry

docs/                      consumer documentation for the SDK and integrations
examples/                  runnable integration examples
test/                      top-level regression tests across packages
```

---

## Module Organization

### Put new code in `cmd/server` when

- it is process boot logic
- it chooses runtime mode from flags or environment
- it selects the persistence backend or startup path

### Put new code in `internal/api` when

- it is Gin server setup, middleware, or management-specific routing
- it should only exist in the bundled server
- it depends on server-local concerns such as management auth or static asset serving

### Put new code in `sdk/api` or `sdk/cliproxy` when

- embedders should get the same behavior as the server binary
- the logic is part of protocol handling, runtime auth selection, or service lifecycle
- consumers might reasonably import it from Go code

### Put new code in `internal/api/modules` when

- the feature is an optional route bundle that should plug into the server without bloating core route registration

### Put new code in `internal/store` when

- it changes how config or auth state is mirrored to file/Postgres/Git/Object storage
- it should not leak store-specific concerns into handlers or provider code

---

## Naming Conventions

- Use lowercase package directories.
- Follow the existing descriptive filename patterns:
  - `*_test.go` for tests
  - `*_handlers.go` for protocol handlers
  - `*_login.go` for provider auth/login flows
  - `*_store.go` or `*store.go` for persistence backends
- Keep package names short and behavior-based (`logging`, `watcher`, `usage`, `store`).
- Prefer adding focused files beside related code over creating large “misc” catch-all files.

---

## Examples

- Route registration and management gating: `internal/api/server.go`
- Embeddable service construction: `sdk/cliproxy/builder.go`
- Runtime lifecycle and watcher queue handling: `sdk/cliproxy/service.go`
- Optional store implementations: `internal/store/postgresstore.go`, `internal/store/gitstore.go`, `internal/store/objectstore.go`
