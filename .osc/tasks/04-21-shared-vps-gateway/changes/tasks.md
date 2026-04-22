# Tasks: Move CLIProxyAPI Deploy to Shared VPS Gateway

- Date: 2026-04-21
- Owner(s): hewei
- Related: spec.md, proposal.md

## Assumptions

- The VPS shared gateway root is `/opt/vps-gateway`.
- The shared gateway container is `vps-gateway-nginx`.
- The shared Docker network is `vps-gateway`.
- The local Claude-compatible upstream container `kiro-rs` is attached to `cli-proxy-api-proxy`.

## Checklist

- [x] 1) Update production compose
  - Target: `deploy/compose.production.yml`
  - Change: remove nginx service; make `proxy` external `vps-gateway`.
  - Verify: compose config validation.

- [x] 2) Update deployment script
  - Target: `deploy/scripts/remote-deploy.sh`
  - Change: remove cert checks, create/use gateway network, install route config into gateway conf dir, validate/reload gateway container.
  - Verify: `bash -n deploy/scripts/remote-deploy.sh`.

- [x] 3) Update env/docs
  - Target: `deploy/.env.example`, `deploy/README.md`
  - Change: add gateway vars and document shared entrypoint topology.
  - Verify: docs review.

- [x] 4) Run quality gates
  - Target: deploy config and Go build.
  - Change: no Go behavior change.
  - Verify: `go build -o test-output ./cmd/server`, compose config, shell syntax.

- [x] 5) Persist split-proxy local upstream network
  - Target: `deploy/compose.production.split-proxy.yml`, `deploy/scripts/remote-deploy.sh`
  - Change: attach `split-proxy` to `${LOCAL_CLAUDE_NETWORK:-cli-proxy-api-proxy}` in addition to the gateway network, and validate that the network exists before deploy.
  - Verify: production compose config includes both networks and `bash -n deploy/scripts/remote-deploy.sh` passes.

- [x] 6) Update split-proxy docs and env defaults
  - Target: `deploy/.env.example`, `deploy/README.md`, `deploy/SPLIT_PROXY_SETUP_CN.md`, split-proxy compose defaults.
  - Change: document `LOCAL_CLAUDE_NETWORK`, include `kiro-rs` in direct bypass host defaults, and retain `kirors-kiro` compatibility.
  - Verify: docs review and compose config interpolation.

## Notes

- VPS runtime was already migrated manually to `/opt/vps-gateway`; this change prevents future repo deployments from reverting that migration.
- Checks passed on 2026-04-21: Go build, production compose config with and without split-proxy, shell/YAML syntax, remote deploy script run on `bytevirt`, gateway nginx test.
- On 2026-04-22, runtime logs showed Squid 502 because `cli-proxy-split-proxy` was only attached to `vps-gateway` while `kiro-rs` was only attached to `cli-proxy-api-proxy`.
- Checks passed on 2026-04-22: Go build, production compose config with and without split-proxy, local split-proxy compose config, and remote deploy script syntax.
