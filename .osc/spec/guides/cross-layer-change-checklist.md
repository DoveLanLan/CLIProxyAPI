# Cross-Layer Change Checklist

Use this checklist when a change crosses server routes, SDK behavior, config persistence, auth storage, or management/TUI surfaces.

## 1. Surface Classification

- Is the change public API compatibility (`/v1`, `/v1beta`) or management-only (`/v0/management`)?
- Does the behavior also need to work in the embeddable SDK?
- Is any UI affected in the TUI or external management clients?

## 2. Persistence Impact

- Does the change alter `config.yaml` shape or semantics?
- Does it affect auth file layout or store mirroring?
- If config is written through management handlers, does validation still happen before persistence?

## 3. Runtime Impact

- Does the watcher/hot-reload path still apply changes without restart?
- Does the auth selector/scheduler behavior still make sense after the change?
- Are request retries, quota cooling, or model aliasing affected?

## 4. Error And Logging Impact

- Are public API errors still protocol-compatible?
- Are management errors still operator-readable?
- Could logs or error payloads now expose secrets, cookies, or tokens?

## 5. Release And Review Impact

- Does `go build ./cmd/server` still pass?
- Is `internal/translator/**` untouched?
- If the change affects packaging or startup, are Docker and GoReleaser assumptions still valid?
- If the change is risky, has the rollback path been written into the task change package?
