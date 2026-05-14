# PRD: Integrate CPA-Manager panel and usage monitoring

## Problem
CPA-Manager can replace the current management panel, but request monitoring needs a Usage Service and a CPA usage queue endpoint that this branch did not expose.

## Goals
- Serve CPA-Manager as the default `/management.html` panel.
- Add minimal HTTP usage queue compatibility for CPA-Manager Usage Service.
- Document local and production deployment with persistent Usage Service storage.

## Non-goals
- Do not vendor CPA-Manager source.
- Do not perform a full upstream version upgrade.
- Do not expose management endpoints publicly.

## Acceptance criteria
- CPA defaults to `https://github.com/seakee/CPA-Manager` for panel release assets.
- `GET /v0/management/usage-queue?count=N` pops queued usage records under existing management auth.
- `GET /v0/management/api-key-usage` returns provider/API-key grouped recent request stats.
- `go test ./...` and `go build -o test-output ./cmd/server && rm test-output` pass.
