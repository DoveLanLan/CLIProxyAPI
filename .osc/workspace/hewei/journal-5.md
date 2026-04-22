# hewei journal 5

- Date: 2026-04-22
- Title: Remote split-proxy validation complete
- Commit: 7b24bbef

## Summary

Remote post-deploy validation for the shared VPS gateway and split-proxy network fix completed successfully.

Validation evidence provided from `root@t7y08hlk8c`:

```text
docker exec vps-gateway-nginx nginx -t
nginx: the configuration file /etc/nginx/nginx.conf syntax is ok
nginx: configuration file /etc/nginx/nginx.conf test is successful

docker exec cli-proxy-split-proxy getent hosts kiro-rs
172.18.0.4      kiro-rs
```

Conclusion: the shared gateway nginx config remains valid after deploy, and `cli-proxy-split-proxy` can resolve the local Claude-compatible upstream service `kiro-rs` through the configured local Docker network.

Remaining follow-up: watch the GitHub Actions production workflow associated with commit `7b24bbef` if deployment status needs to be tied back to CI.
