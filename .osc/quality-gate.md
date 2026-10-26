# Quality Gate: Sync Upstream v8.0.43

- Date: 2026-10-26
- Task: `.osc/tasks/10-26-sync-upstream-v8`
- Branch: `task/sync-upstream-v8` → `main`
- Rollback point: `cf5070da`

## Gate results

| Check | Command | Result |
|---|---|---|
| gofmt | `gofmt -l .` | pass (0 files) |
| Build | `go build -o test-output ./cmd/server` | pass |
| Vet (incl. test compile) | `go vet ./...` | pass |
| Tests | `go test ./...` | pass — 103 packages ok, 0 failures |
| G1 module path | no `CLIProxyAPI/v7` in `*.go` | pass (0) |
| G2 protected paths | `git diff cf5070da -- .github Dockerfile .dockerignore docker-build.* docker-compose*.yml .env.cluster.example deploy .goreleaser.yml` | pass (empty) |
| G3 CPA-Manager-Plus | grep in config defaults + updater + example | pass (4 hits) |
| Conflict markers | grep `^<<<<<<<` / `^>>>>>>>` | pass (0) |
| Unresolved paths | `git diff --name-only --diff-filter=U` | pass (0) |

## Scope

- Module path v7 → v8 (upstream migration + 25 fork-only files).
- `go 1.26.10` toolchain, new Pion/WebRTC + zeroconf dependencies.
- Upstream `v8.0.43` absorbed (1441 commits, 1810 files), including 797 Antigravity-touching commits.
- Fork customizations preserved: deploy/CI, CPA-Manager-Plus, xAI pool + Resin, Command Code, DeepSeek reasoning echo, tool-result images, thinking overrides, Claude error metadata.
- Superseded fork patches dropped (see change summary §4).

## Risks accepted

1. **Go 1.26.10 toolchain fetch** — `GOTOOLCHAIN=auto` downloads it on first build. Verified locally.
2. **CI build time** — new WebRTC/Pion dependency tree increases image build time; no config change needed.
3. **`gpt-5.5` returned to codex-free** — intended upstream behavior; documented revert in rollback notes.
4. **Antigravity behavior drift** — 155 antigravity-named upstream changes alter tool-signature replay, compaction, connections. Covered by upstream tests; production smoke checks required post-deploy.

## Sign-off

- [x] Changes documented (`proposal.md`, `spec.md`, `tasks.md`, `change-summary.md`)
- [x] Regression checklist written (`regression-checklist.md`)
- [x] Rollback notes written (`rollback-notes.md`)
- [x] All automated gates green
- [ ] Post-deploy production smoke checks (`regression-checklist.md` §Manual)
