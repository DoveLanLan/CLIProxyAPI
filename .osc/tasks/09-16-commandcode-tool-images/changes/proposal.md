# Command Code tool image compatibility

Production eb923914 forwards images, but request f1031670 fails at Command Code
with `Invalid input`: a Claude Read result became an image_url inside a Chat tool
message. Prior mocks accepted this invalid role/content combination.

Normalize Chat tool image results in the OpenAI-compatible executor, preserving
tool IDs and text while moving images into a user message after the complete
contiguous tool-result group. Do not change the shared translators, model routing,
timeouts, or credentials. Verify using Claude Code CLI against production.

Risk: moving images changes message layout. Preserve order, annotate originating
tool IDs, and test multiple parallel tool results and mixed text/image content.
