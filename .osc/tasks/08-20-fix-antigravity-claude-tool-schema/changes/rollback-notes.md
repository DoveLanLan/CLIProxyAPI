# Rollback Notes

## Bytevirt

Restore the backed-up binary and restart only `cli-proxy-api`:

```bash
docker cp /opt/cliproxyapi/CLIProxyAPI.bak-20260820-033222 cli-proxy-api:/CLIProxyAPI/CLIProxyAPI
docker restart cli-proxy-api
```

The repository change can be reverted independently. No config, auth, database, or model-registry migration was performed.

## Safety

The backup is the binary that was running before this hotfix. Keep it until the next image deployment is confirmed healthy.
