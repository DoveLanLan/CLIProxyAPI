# Proposal: harden Claude Code DeepSeek continuations

## Motivation

Long Claude Code sessions are interrupted by several distinct 400-class failures:
missing DeepSeek reasoning replay, text-only image rejection, and generic upstream
bootstrap failures. Small tool histories are valid, so disabling all reasoning or all
parallel tools would be unnecessarily broad.

## Proposed changes

- Recover original Claude thinking for DeepSeek assistant tool calls after protocol
  translation and before strict OpenAI message-link normalization.
- Add focused unit tests without weakening the shared signature compatibility policy.
- Apply DeepSeek-only local compatibility settings and an image-Read guard.
- Enable one safe production streaming bootstrap retry after backing up config.
- Preserve safe upstream error details through the Claude-compatible response path
  instead of collapsing them to an unexplained generic 400.
- Reject known DeepSeek text-only image input and unsupported named `tool_choice`
  locally with a structured compatibility error before forwarding upstream.
- Rotate the upstream credential exposed during diagnosis, then deliver the verified
  source patch through the immutable production workflow.

## Out of scope

- Modifying the Claude Code binary.
- Disabling reasoning globally.
- Claiming the configured `[1m]` suffix proves a one-million-token upstream limit.
- Retrying after response payload bytes or tool side effects have been emitted.
- Inventing an upstream context limit that the provider does not publish.
