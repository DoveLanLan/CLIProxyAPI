# Requirements

- Claude-origin DeepSeek image requests must reach the configured Chat endpoint.
- Base64 PNG images must become `image_url` parts with the original data URL.
- Cover direct image blocks and nested tool-result images, streaming and non-streaming.
- Do not infer vision capability from model prefixes, including provider prefixes.
- Preserve named-tool rejection and existing reasoning echo behavior.
- Preserve upstream HTTP image rejection rather than replacing it locally.
- No public configuration or schema migration; upstream image errors may differ
  from the former local `model_text_only` response.
