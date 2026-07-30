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

The pre-change VPS file is:

`/opt/cliproxyapi/data/backups/config.yaml.deepseek-tool-400.20260730T034541Z`

Restore both the host file used by the next container recreation and the currently
mounted container inode, allowing the watcher to reload the old value:

```bash
ssh bytevirt 'set -eu
cp --preserve=mode,ownership \
  /opt/cliproxyapi/data/backups/config.yaml.deepseek-tool-400.20260730T034541Z \
  /opt/cliproxyapi/data/config.yaml
docker exec -i cli-proxy-api sh -c "cat > /CLIProxyAPI/config.yaml" \
  < /opt/cliproxyapi/data/backups/config.yaml.deepseek-tool-400.20260730T034541Z
'
```

No immediate restart is required. A later normal recreation will remount the host
file and eliminate the temporary inode difference.

## Repository source patch

Before a commit, review the patch with `git diff` and restore only these tracked
files if the candidate gateway fix must be discarded:

```bash
git restore -- \
  internal/runtime/executor/helps/openai_message_links.go \
  internal/runtime/executor/openai_compat_executor.go \
  internal/runtime/executor/openai_compat_executor_compact_test.go \
  sdk/api/handlers/claude/code_handlers.go \
  sdk/api/handlers/claude/code_handlers_error_test.go
rm -f \
  internal/runtime/executor/helps/openai_compat_deepseek.go \
  internal/runtime/executor/helps/openai_compat_deepseek_test.go \
  internal/runtime/executor/helps/openai_message_links_test.go
```

After deployment, roll back through the normal immutable-image workflow by reverting
the eventual patch commit and deploying the resulting image. Do not edit the binary
inside the running container.
