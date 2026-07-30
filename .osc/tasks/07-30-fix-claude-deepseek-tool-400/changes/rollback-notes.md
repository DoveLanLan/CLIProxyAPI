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
git revert --no-edit 5bf14e5d21925dbd915336cf37ac0f3b46aeb20e
git revert --no-edit aa2cbf95f8eb6906399fd0c74b316ee51a6ef622
git push origin main
```

Wait for the resulting Docker and production workflows, then verify the new OCI
revision and `/healthz`. Do not edit the binary inside the running container.
