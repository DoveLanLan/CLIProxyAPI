# Database Guidelines

> Persistence rules for configuration and auth state in CLIProxyAPI.

---

## Overview

This project does not revolve around an application ORM schema. Its persisted state is mainly:

- `config.yaml`
- auth/token files under the configured auth directory
- optional mirrored backends for config/auth state
- usage/log files as operational artifacts

The important rule is to preserve the file-oriented workflow even when backing storage is remote.

---

## Persistence Model

### Default mode

By default, the server works from local files:

- config lives in a YAML file
- auth records live in `auth-dir`
- management endpoints mutate the in-memory config and persist it back to disk

Primary evidence: `config.example.yaml`, `internal/api/handlers/management/config_basic.go`, `internal/api/handlers/management/handler.go`

### Optional backing stores

The server can bootstrap alternative config/auth persistence backends:

- PostgreSQL via `internal/store/postgresstore.go`
- Git-backed state via `internal/store/gitstore.go`
- S3-compatible object storage via `internal/store/objectstore.go`

Each backend mirrors data into a local workspace so existing file-based code paths continue to work.

Primary evidence: `cmd/server/main.go`, `internal/store/postgresstore.go`, `internal/store/gitstore.go`, `internal/store/objectstore.go`

---

## Query Patterns

- There is no ORM in the shallow repo inventory.
- PostgreSQL access is localized to the store layer and uses `database/sql` with `pgx` driver compatibility.
- Store implementations are responsible for:
  - preparing local directories
  - syncing remote state to local files
  - persisting updates to both local and remote storage
  - avoiding redundant rewrites when content is unchanged

Do not spread SQL, Git sync logic, or S3 client logic into handlers or protocol code.

---

## Migrations

- No standalone migration framework was found.
- PostgreSQL schema bootstrap is done in-process via `EnsureSchema`.
- The store layer owns schema/table creation for its own config/auth tables.

Implication: if you extend a persistence backend, keep schema/bootstrap logic close to that backend instead of introducing a parallel migration system without a broader design change.

---

## Naming Conventions

- Treat config as a single managed document, not as many unrelated database rows.
- Keep auth storage keyed by auth identity / file path semantics that still make sense when mirrored back to disk.
- When a store mirrors files locally, preserve the `config/` and `auths/` shape rather than inventing a backend-only layout that the rest of the code cannot understand.

---

## Common Mistakes

- Do not introduce a new “real database layer” for request-path behavior when file/config-backed state is the existing contract.
- Do not bypass the local mirror when implementing a new backend; watcher and management flows assume file-compatible state still exists.
- Do not write config with generic YAML serialization if comment-preserving persistence is required.
- Do not mix storage concerns into protocol handlers or TUI code.
