# Journal 12: CPA-Manager Fork Panel Release Handoff

- Date: 2026-05-22
- Related document: `docs/handoff/cpa-manager-fork-panel-release.md`

## Conclusion

`remote-management.panel-github-repository` does not control the external CPA-Manager Docker image. It controls where CLIProxyAPI downloads its built-in `/management.html` panel asset from GitHub Releases.

The VPS external CPA-Manager service already uses the fork image:

- `ghcr.io/dovelanlan/cpa-manager:sha-7fa4bfb77b917ddd02141b7fd723182cf2a47013`

The fork does not yet expose a latest GitHub Release with `management.html`, so switching CLIProxyAPI `panel-github-repository` to the fork now may break built-in panel downloads.

## Next Step

Add or adapt a CPA-Manager fork release workflow so `DoveLanLan/CPA-Manager` publishes `management.html` as a GitHub Release asset. After the release is verified, switch CLIProxyAPI `panel-github-repository` to `https://github.com/DoveLanLan/CPA-Manager`.

## Risks

- Existing upstream `release.yml` also tries to publish Docker Hub images and may fail in the fork without Docker Hub secrets.
- `panel-github-repository` changes only CLIProxyAPI's built-in panel; it does not affect the external CPA-Manager service on port `18318`.

## Rollback

Keep or restore:

```yaml
remote-management:
  panel-github-repository: "https://github.com/seakee/CPA-Manager"
```

This rollback is independent from the external `CPA_MANAGER_IMAGE` pin.
