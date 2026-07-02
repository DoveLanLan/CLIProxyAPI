# Handoff: CPA-Manager-Plus Panel Release

- Date: 2026-05-22
- Owner: hewei
- Repositories:
  - CLIProxyAPI: `https://github.com/DoveLanLan/CLIProxyAPI`
  - CPA-Manager fork: `https://github.com/DoveLanLan/CPA-Manager`

## Current State

- The VPS external CPA-Manager service on port `18318` already uses the fork image:
  - `ghcr.io/dovelanlan/cpa-manager:sha-7fa4bfb77b917ddd02141b7fd723182cf2a47013`
- CLIProxyAPI has a workflow that updates the VPS `.env` key `CPA_MANAGER_IMAGE` and restarts only the `cpa-manager` service:
  - `.github/workflows/update-cpa-manager-image.yml`
- The monitoring page at `http://100.67.99.9:18318/management.html#/monitoring` rendered data after the deploy.
- CLIProxyAPI `config.yaml` should point the built-in panel updater at CPA-Manager-Plus:
  - `remote-management.panel-github-repository: "https://github.com/seakee/CPA-Manager-Plus"`

## Why This Still Needs Work

`panel-github-repository` is not the Docker image source. It controls where CLIProxyAPI downloads the built-in `/management.html` panel asset from GitHub Releases.

The Plus repository currently has a latest release with `management.html`:

- `seakee/CPA-Manager-Plus` latest release: `v1.10.1` (verified 2026-07-02)
- Asset includes: `management.html`

The fork does not currently have a latest release, so changing CLIProxyAPI to:

```yaml
panel-github-repository: "https://github.com/DoveLanLan/CPA-Manager"
```

before publishing a fork release may make CLIProxyAPI's built-in panel download fail.

## Goal

Make CLIProxyAPI's built-in management panel use the CPA-Manager fork safely:

1. Publish `management.html` from `DoveLanLan/CPA-Manager` GitHub Releases.
2. Verify the fork release API exposes `management.html`.
3. Change CLIProxyAPI `panel-github-repository` to `https://github.com/DoveLanLan/CPA-Manager`.
4. Deploy/verify CLIProxyAPI's built-in `/management.html` still loads.

## Recommended Plan

### 1. Prepare Both Repositories

```bash
cd /path/to/workspace
git clone https://github.com/DoveLanLan/CPA-Manager.git
git clone https://github.com/DoveLanLan/CLIProxyAPI.git
```

If already cloned:

```bash
cd CPA-Manager
git checkout main
git pull --ff-only origin main

cd ../CLIProxyAPI
git checkout main
git pull --ff-only origin main
```

### 2. Add a Fork-Safe Release Workflow in CPA-Manager

Target repository: `DoveLanLan/CPA-Manager`

The upstream `.github/workflows/release.yml` already builds `dist/index.html` and renames it to `dist/management.html`, but it also has a Docker Hub publishing job that expects upstream Docker Hub secrets.

For the fork, prefer one of these approaches:

- Option A: add a new workflow, for example `.github/workflows/panel-release.yml`, that only builds and releases `management.html`.
- Option B: edit `.github/workflows/release.yml` so the Docker job publishes to GHCR or is disabled for the fork.

Recommended minimal workflow behavior:

- Trigger on `v*` tags and `workflow_dispatch`.
- Use Node 24.
- Run `npm ci`.
- Run `VERSION=${GITHUB_REF_NAME:-manual} npm run build`.
- Copy `dist/index.html` to `dist/management.html`.
- Create or update a GitHub Release with `dist/management.html`.

Validation commands before committing:

```bash
npm run type-check
npm run lint
VERSION=ci npm run build
npm test
git diff --check
```

### 3. Publish a Fork Release

Use a fork-specific tag so it is clear this is not upstream:

```bash
git tag v1.3.3-dovelanlan.1
git push origin v1.3.3-dovelanlan.1
```

After the workflow finishes, verify:

```bash
curl -fsSL https://api.github.com/repos/DoveLanLan/CPA-Manager/releases/latest \
  | jq -r '.tag_name, (.assets[]?.name)'
```

Expected output should include:

```text
v1.3.3-dovelanlan.1
management.html
```

Also verify the direct asset URL works:

```bash
curl -fI -L \
  https://github.com/DoveLanLan/CPA-Manager/releases/latest/download/management.html
```

### 4. Change CLIProxyAPI Panel Source

Target repository: `DoveLanLan/CLIProxyAPI`

Change any production config/documentation that still points to upstream when the intention is to use the fork panel:

```yaml
remote-management:
  panel-github-repository: "https://github.com/DoveLanLan/CPA-Manager"
```

Likely files to inspect:

- `config.yaml` if present locally or on the VPS
- `config.example.yaml`
- `deploy/README.md`
- any deployment templates that mention `panel-github-repository`

Important: avoid committing real secrets from `config.yaml`.

Validation commands:

```bash
git diff --check
go build -o test-output ./cmd/server && rm test-output
```

Run broader tests if code changes beyond docs/config examples:

```bash
go test ./...
```

### 5. Deploy and Verify

After CLIProxyAPI is deployed with the fork panel repository:

1. Restart or redeploy CLIProxyAPI so it reads the updated config.
2. If the old `management.html` is cached under the server static asset directory, remove it or use the built-in updater path to fetch the new release.
3. Open CLIProxyAPI's built-in panel, not the external `18318` CPA-Manager panel:
   - `http://100.67.99.9:18317/management.html#/`
4. Confirm it loads from the fork release and does not regress login/config/monitoring.

Keep verifying the external CPA-Manager service separately:

- `http://100.67.99.9:18318/health`
- `http://100.67.99.9:18318/management.html#/monitoring`

## Risks

- Fork release missing `management.html`: CLIProxyAPI built-in panel update can fail.
- Fork release workflow accidentally keeps the upstream Docker Hub publishing job: tag release can fail due to missing Docker Hub secrets.
- Switching `panel-github-repository` affects CLIProxyAPI's built-in panel only; it does not update the external CPA-Manager Docker image.
- Browser or server cache may keep showing an older `management.html` until the local asset is refreshed.

## Rollback

Set CLIProxyAPI back to upstream panel releases:

```yaml
remote-management:
  panel-github-repository: "https://github.com/seakee/CPA-Manager-Plus"
```

Then restart/redeploy CLIProxyAPI and refresh the local panel asset.

The external CPA-Manager service can stay pinned to the fork image through `CPA_MANAGER_IMAGE`; this rollback only affects CLIProxyAPI's built-in `/management.html`.

## Useful References

- CPA-Manager fork image currently deployed:
  - `ghcr.io/dovelanlan/cpa-manager:sha-7fa4bfb77b917ddd02141b7fd723182cf2a47013`
- CPA-Manager fork commit containing monitoring performance fix:
  - `7fa4bfb77b917ddd02141b7fd723182cf2a47013`
- CLIProxyAPI ops commit that deployed that image:
  - `d36a2b48ba2bceddc4e657a7fccb8772b0a38e7f`
