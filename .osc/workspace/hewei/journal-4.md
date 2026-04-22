# hewei journal 4

- Date: 2026-04-22
- Title: Commit shared VPS split-proxy network fix
- Commit: 

## Summary
Conclusions/decisions: production split-proxy now joins both the shared gateway network and LOCAL_CLAUDE_NETWORK so Docker service upstreams such as kiro-rs resolve after redeploys. The deploy script fails early if LOCAL_CLAUDE_NETWORK is missing when ENABLE_SPLIT_PROXY=true. Direct-bypass defaults now include both kiro-rs and legacy kirors-kiro.

Verification: reran docker compose production config, production split-proxy config, local split-proxy config with sample upstream env, bash -n deploy/scripts/remote-deploy.sh, and go build -o test-output ./cmd/server; all passed and test-output was removed.

Next steps: after push, watch the GitHub Actions production workflow and run remote checks after deployment: docker exec vps-gateway-nginx nginx -t and docker exec cli-proxy-split-proxy getent hosts kiro-rs.

Risks/rollback: residual risk is remote LOCAL_CLAUDE_NETWORK value or service name drift; rollback by reverting the local-claude network block and LOCAL_CLAUDE_NETWORK docs/env update, then detaching cli-proxy-split-proxy from cli-proxy-api-proxy if needed.
