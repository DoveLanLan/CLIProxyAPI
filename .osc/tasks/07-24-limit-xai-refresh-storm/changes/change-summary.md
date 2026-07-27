# Change Summary: Limit xAI Credential Refresh Storm

- Date: 2026-07-24
- Production image: `cliproxyapi:xai-refresh-fix-20260724`
- Base revision: `40b33cb1`

## Implemented

- Disabled credentials are excluded from both automatic refresh evaluation and scheduler insertion.
- Explicit xAI `invalid_grant` token refresh responses are exposed as unauthorized failures, using the manager's existing permanent in-process suspension behavior.
- Production automatic refresh concurrency is capped at two workers.
- Production transient 408/500/502/503/504 credential cooldown is set to 300 seconds.
- Existing request bounds remain `request-retry: 2` and `max-retry-credentials: 4`.
- Existing SSE keepalive remains 15 seconds.

## Production Verification

- Service and private management endpoint returned healthy status.
- Initial refresh pass touched zero disabled credentials.
- The one enabled credential returning `invalid_grant` was attempted once and did not reappear after a complete five-minute legacy retry interval.
- Subsequent scheduled refreshes were limited to enabled credentials.
- A real Claude-compatible Grok stream using `claude-opus-4-8` completed with HTTP 200 and `message_stop`.
- Gateway traffic after the verification cutoff contained successful `/v1/messages` responses and no 429/500/502/503/504/524 response.

## Operational Note

One request arrived in the same second as the planned container replacement and received a one-time 502. The pre-deploy idle check had shown no established or recent request, but the new request raced with the restart.
