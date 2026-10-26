# Rollback Notes: Sync Upstream v8.0.43

- Date: 2026-10-26
- Pre-sync rollback point: `cf5070da` (`v7.2.93-94-gcf5070da`, local `main` before this sync)

## Branches

- Work branch: `task/sync-upstream-v8` (merge commit of `cf5070da` + `upstream/main` @ `v8.0.43`)
- `main` fast-forwards to that merge commit.

## Rollback before deploy completes

```bash
# On the machine that pushed
git checkout main
git reset --hard cf5070da
git push --force-with-lease origin main
```

The `docker-image` workflow is keyed on `main`; a revert push rebuilds and redeploys from the reverted tree. `deploy-production` runs off `workflow_run` of `docker-image` for `head_branch == main`, and is serialized by `concurrency: production` with `cancel-in-progress: true`, so the newer run wins.

## Rollback after deploy (production host)

The deploy pulls a GHCR image tag built from `main`. Roll back by pinning the previous image tag:

```bash
cd /opt/cliproxyapi
./deploy/scripts/remote-deploy.sh --image-tag <previous-tag>   # or edit .env and re-run compose pull
docker compose -f deploy/compose.production.yml up -d
```

Previous tag: the last `main` build before this sync (the commit right before `cf5070da`'s descendant merge, i.e. `cf5070da` itself).

Alternative: trigger `deploy-production` manually via `workflow_dispatch` with `image_tag=<previous-tag>`.

## Reverting individual decisions (no full rollback needed)

| Decision | One-line revert |
|---|---|
| `gpt-5.5` re-added to `codex-free` | delete the `gpt-5.5` object from the `codex-free` array in `internal/registry/models/models.json` |
| Codex invalidated-token now recovers via refresh instead of hard-disable | restore `applyCodexOAuthTokenInvalidatedState` + guards in `sdk/cliproxy/auth/conductor_{cooldown,stream,execution,refresh}.go` (see `cf5070da` for the original code) |
| Pre-first-token keepalive now opt-in | set `streaming.send-keepalive-before-ft: true` in `config.yaml` alongside `keepalive-seconds` |
| Fork config sections preserved on v8 writes | remove `xai-proxy-pool` / `xai-resin-proxy` / `commandcode-api-key` from `v8AllowedRoots()` in `internal/config/config_v8.go` (only needed if the fork sections are dropped) |

## Data/config safety

- `config.yaml`, `auths/`, `logs/`, `plugins/` are gitignored and were not part of the sync commit.
- The v8 config migration is **write-triggered**: loading a legacy `config.yaml` does not rewrite it; only a successful `/v8/management` write migrates. The fork's sections are registered so they survive that write.
- No secrets were added to the repository.

## Verification of a rollback

After any rollback, confirm:

```bash
go build ./cmd/server          # or the container image pulled successfully
go test ./internal/config/ ./sdk/cliproxy/auth/
curl -sS localhost:8317/v1/models | head
grep -E 'xai-proxy-pool|xai-resin-proxy|commandcode-api-key' config.yaml
```
