# Rollback Notes

## Disable the Feature

Remove the `commandcode-api-key` entries from the deployed YAML configuration. The
existing watcher removes those credentials from routing; other providers remain
unchanged. There are no database migrations, persistent provider state files, extra
listeners or sidecar containers to clean up. In-flight requests follow existing
request cancellation/lifecycle behavior.

## Roll Back a Future Deployment

Remove the new configuration first, then deploy the previous known-good CPA image
using the existing immutable-image deployment procedure. Do not change unrelated
provider credentials or overwrite the entire server configuration.

## Workspace State

This task has not committed, pushed or deployed changes. Source changes are isolated
to the new provider and its config/watcher/service/alias hooks, the model-discovery
flag, documentation and OSC artifacts. No rollback action has been executed.
