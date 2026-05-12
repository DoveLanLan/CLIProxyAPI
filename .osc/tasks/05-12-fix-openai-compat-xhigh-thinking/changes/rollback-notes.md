# Rollback Notes: Fix OpenAI Compat XHigh Thinking Preservation

- Date: 2026-05-12
- Related: spec.md, tasks.md

## Rollback Procedure

- Revert the changes in `sdk/cliproxy/service.go`, `sdk/cliproxy/service_openai_compat_thinking_test.go`, and `config.example.yaml`.
- No database, storage, auth, or config migration rollback is required.

## Operational Impact

- Rolling back restores the previous OpenAI-compatible default levels of `low`, `medium`, and `high`, which can again clamp `xhigh` to `high` when no explicit model `thinking` metadata is configured.

## Safer Alternative To Full Rollback

- If a specific upstream rejects `xhigh`, add explicit per-model config:

```yaml
thinking:
  levels: ["low", "medium", "high"]
```

This preserves the safer generic default for other OpenAI-compatible upstreams.
