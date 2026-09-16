# Summary

Removed the DeepSeek-wide image preflight rejection and its unused recursive
scanner. Images now pass through the existing Claude-to-Chat translation; model
capabilities are validated by the selected upstream. Named-tool validation and
reasoning recovery remain unchanged.

Eight executor regression cases cover streaming/non-streaming, direct/tool-result
PNG images, and bare/provider-prefixed V4.1 Flash model names. Helper tests cover
legacy and future names; an executor test verifies upstream image error retention.

No translator, configuration, dependency, timeout, or production changes.
Existing text-only deployments may return upstream-specific image errors instead
of CPA's synthetic `model_text_only` error. Live Command Code acceptance remains
unverified until an authorized deployment and smoke test.
