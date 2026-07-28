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

After production rollout, add one same-auth retry for a Resin transport failure
that occurs before any response payload is exposed downstream. Resin invalidates
the failed Account lease synchronously, so rebuilding the request with the same
derived identity reaches the replacement node without switching xAI credentials.

## Compatibility

- Disabled by default.
- Existing global and per-auth proxy behavior is unchanged while disabled.
- Explicit per-auth proxies always bypass automatic Resin routing.
- Existing EgressProxyPool support remains, but simultaneous enablement fails
  closed instead of choosing an ambiguous backend.
- No public API or translator changes.
- The retry is internal, limited to one replay, and never selects another auth,
  EgressProxyPool route, or CPA global proxy.

## Security

- Secret values are read from files and never stored in the main config.
- Raw auth IDs are never sent to Resin.
- Proxy URL construction uses structured URL userinfo encoding.
- Errors and logs do not include tokens, identity keys, or credentialed URLs.

## Rollback

Disable or remove `xai-resin-proxy` and remove the secret mounts. No auth-file
migration or persisted-state rollback is needed.

## Official delivery correction

The production follow-up must return both repositories to their existing
GitHub Actions delivery path. Local Docker tags and image transfer are not a
release mechanism. Resin must publish and deploy
`ghcr.io/dovelanlan/resin:sha-<commit>` from `master`; CPA must publish and
deploy `ghcr.io/dovelanlan/cliproxyapi:sha-<commit>` from `main`.

CPA's deploy script currently sources the server `.env` after the workflow has
provided `CLI_PROXY_IMAGE`, allowing the stale file value to replace the
workflow-selected immutable image. Preserve an explicitly supplied image across
`.env` loading, while retaining `.env` as the fallback for interactive server
deployments. The correction must be covered by an executable regression test.

Before pushing CPA, integrate the remote exact-402 Resin lease-rotation change
with the local pre-response network retry. Both retry classes must remain
bounded and independently safe. Push Resin first, wait for its automatic
deployment and health verification, then push CPA and wait for the same gates.

## Authorized production rollout

On 2026-07-28 the operator explicitly authorized auditing and completing the
deployment on the `bytevirt` VPS. The rollout may update `/opt/cliproxyapi` and
`/opt/resin`, create the two required CPA secret files, replace the CPA container
through the repository workflows, and verify real Resin and xAI traffic.
Existing files must be backed up before mutation and rollback must remain
available until end-to-end verification passes.
