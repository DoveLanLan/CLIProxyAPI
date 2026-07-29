# Tech notes

- Keep Resin's synchronous first-meaningful-payload classification unchanged;
  bridge the wait at the downstream SSE handler with a joined keep-alive writer.
- Reuse `streaming.keepalive-seconds`; do not add another runtime configuration
  key or any post-connect upstream timeout.
- Stop and join the bootstrap writer before the handler resumes writing to the
  response, avoiding concurrent writes.
- Once a heartbeat commits the response, later startup failures must be encoded
  as SSE errors because the HTTP status can no longer change.
- Keep the Grok timer switch deployment-only. Default it to `true` for existing
  installations, while production explicitly sets it to `false`.
- Roll back bootstrap behavior by reverting the handler/helper changes. Roll
  back timer behavior by setting `ENABLE_GROK_INSPECTION_TIMER=true`.
