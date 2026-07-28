# Proposal: native Resin routing for xAI

## Motivation

Resin provides sticky routing when clients authenticate to its forward proxy as
`Platform.Account:token`. CPA already knows the selected xAI auth ID, but its
static proxy configuration cannot turn that ID into a different proxy username
per request. Per-auth configuration does not scale to large credential pools.

## Proposed change

Add an optional `xai-resin-proxy` config block containing a Resin forward-proxy
URL, Platform name, proxy-token file, and CPA-only identity-key file. At runtime,
the xAI auto executor derives `xai-<HMAC(identity-key, auth-id)>`, clones the
selected auth in memory, and installs a credentialed Resin proxy URL on the
clone. Existing provider transports then apply it uniformly.

## Compatibility

- Disabled by default.
- Existing global and per-auth proxy behavior is unchanged while disabled.
- Explicit per-auth proxies always bypass automatic Resin routing.
- Existing EgressProxyPool support remains, but simultaneous enablement fails
  closed instead of choosing an ambiguous backend.
- No public API or translator changes.

## Security

- Secret values are read from files and never stored in the main config.
- Raw auth IDs are never sent to Resin.
- Proxy URL construction uses structured URL userinfo encoding.
- Errors and logs do not include tokens, identity keys, or credentialed URLs.

## Rollback

Disable or remove `xai-resin-proxy` and remove the secret mounts. No auth-file
migration or persisted-state rollback is needed.

## Authorized production rollout

On 2026-07-28 the operator explicitly authorized auditing and completing the
deployment on the `bytevirt` VPS. The rollout may update `/opt/cliproxyapi` and
`/opt/resin`, create the two required CPA secret files, replace the CPA container
with an image built from this reviewed working tree, and verify real Resin and
xAI traffic. Existing files must be backed up before mutation and rollback must
remain available until end-to-end verification passes.
