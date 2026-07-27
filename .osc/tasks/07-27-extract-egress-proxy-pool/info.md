# Tech notes

- Project name: `EgressProxyPool`.
- The standalone deployment is one Compose project with two containers:
  `egress-proxy-controller` and `mihomo`.
- The control API and proxy listeners remain on a shared private Docker network;
  no host ports are published by default.
- CLIProxyAPI sends only a keyed digest of the auth ID, never the auth ID or
  provider token.
- Probe leases are server-side, single-use, and expire automatically to avoid a
  dead probe selector after client failure.
- Rollback is disabling `xai-proxy-pool` in CLIProxyAPI and restoring the prior
  embedded-pool commit if required. The old state files are not deleted.
