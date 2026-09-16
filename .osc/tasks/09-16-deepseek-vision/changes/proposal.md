# DeepSeek vision preflight fix

## Problem
Production requests 2fdbe4b7, 17fb6270, and c7fcbeaf were rejected locally with
`model_text_only` before Claude-to-Chat translation. The compatibility check
incorrectly assumes every DeepSeek model is text-only.

## Goals and approach
Remove the blanket image rejection and delegate capability validation to the
configured upstream. Verify translated image content for streaming and ordinary
requests, including images nested in tool results. Preserve named-tool choice
validation and reasoning recovery.

## Constraints and non-goals
No translator changes, config changes, new timeouts, deployment, or live billable
requests. Keep the change within executor helpers and executor tests.

## Risks and mitigations
Text-only upstreams will return their own image errors instead of CPA's synthetic
error. Test upstream rejection propagation; do not silently drop image content.
