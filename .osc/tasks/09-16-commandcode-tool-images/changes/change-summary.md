# Summary

The previous fix removed a local image rejection, but actual Read tool results
still failed Command Code's Chat schema. Production request f1031670 contained
an image_url inside tool message 401. Upstream returned HTTP 400 Invalid input.

Added executor-local normalization: keep tool result IDs/text, lift images into
an annotated user message after adjacent tool results. Image URLs/detail and
unrelated request fields remain intact. Streaming/non-streaming use the same
helper. No translator, credentials, configuration, or timeout changes.

Claude Code 2.1.273 reproduced the failure before the change; the patched local
binary forwarding through production to Command Code correctly recognized the
synthetic test card. Direct production acceptance also passed after automated
deployment of bff1543200ec7415e408145860772df82a834f8a. Claude Code Read and a
separate mixed-content parallel-tool test both correctly identified the image.
