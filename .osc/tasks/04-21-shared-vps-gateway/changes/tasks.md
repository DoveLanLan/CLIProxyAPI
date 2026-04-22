# Tasks: Move CLIProxyAPI Deploy to Shared VPS Gateway

- Date: 2026-04-21
- Owner(s): hewei
- Related: spec.md, proposal.md

## Assumptions

- The VPS shared gateway root is `/opt/vps-gateway`.
- The shared gateway container is `vps-gateway-nginx`.
- The shared Docker network is `vps-gateway`.

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

## Notes

- VPS runtime was already migrated manually to `/opt/vps-gateway`; this change prevents future repo deployments from reverting that migration.
- Checks passed on 2026-04-21: Go build, production compose config with and without split-proxy, shell/YAML syntax, remote deploy script run on `bytevirt`, gateway nginx test.
