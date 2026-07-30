# Rollback notes

## Local Claude profile

The pre-change files are under:

`/root/.claude/backups/deepseek-tool-400-20260730T034510Z`

Restore the DeepSeek profile and remove the newly added hook with:

```bash
cp --preserve=mode,ownership,timestamps \
  /root/.claude/backups/deepseek-tool-400-20260730T034510Z/settings.deepseek-pro.json \
  /root/.claude/settings.deepseek-pro.json
rm -f /root/.claude/hooks/deepseek-text-only-read.sh
```

`/root/.claude/settings.json` and `/root/.zshrc` were backed up but not changed.
Restoring them is unnecessary for this task.

## Production bootstrap retry

The full pre-task backup contains the exposed OpenCode credential and must not be
restored wholesale. To roll back only the bootstrap retry while preserving
credential evacuation, replace the single setting in place and let the watcher
reload it:

```bash
ssh bytevirt -- python3 - <<'PY'
path = "/opt/cliproxyapi/data/config.yaml"
with open(path, "r", encoding="utf-8") as handle:
    text = handle.read()
old = "  bootstrap-retries: 1\n"
if text.count(old) != 1:
    raise SystemExit("unexpected bootstrap-retries shape")
with open(path, "w", encoding="utf-8") as handle:
    handle.write(text.replace(old, "  bootstrap-retries: 0\n", 1))
PY
```

Host and container currently share the bind-mounted inode, so no container-side
copy or immediate restart is required.

## Credential containment

The two backups are:

- `/opt/cliproxyapi/data/backups/config.yaml.deepseek-credential-containment.20260730T070256Z`
- `/opt/cliproxyapi/data/backups/config.yaml.deepseek-credential-evacuation.20260730T070555Z`

Both are mode 600 and contain the exposed value. Do not restore either backup to
active config unless service recovery is impossible through the verified fallback
providers, and never restore it after provider-side revocation. Prefer installing a
fresh OpenCode credential through a secure operator session, validating it while the
entry remains disabled, and only then enabling that entry.

## Repository source patch

Roll back through the normal immutable-image workflow with:

```bash
git restore --source=08c59a462ce145116c61d5942dd4f11e61bc4d2a -- \
  internal/runtime/executor/helps/openai_message_links.go \
  internal/runtime/executor/openai_compat_executor.go \
  internal/runtime/executor/openai_compat_executor_compact_test.go \
  sdk/api/handlers/claude/code_handlers.go \
  sdk/api/handlers/claude/code_handlers_error_test.go
git rm -- \
  internal/runtime/executor/helps/openai_compat_deepseek.go \
  internal/runtime/executor/helps/openai_compat_deepseek_test.go \
  internal/runtime/executor/helps/openai_message_links_test.go
gofmt -w .
go test ./...
go build -o test-output ./cmd/server && rm test-output
git commit -m "revert: remove DeepSeek Claude continuation fix"
git push origin main
```

This source-only rollback was dry-run in an isolated worktree and deliberately keeps
the task audit documents. Wait for the resulting Docker and production workflows,
then verify the new OCI revision and `/healthz`. Do not edit the binary inside the
running container.
