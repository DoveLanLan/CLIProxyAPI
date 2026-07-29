# Rollback notes

## Application behavior

Revert the CPA commit containing `bootstrap_keepalive.go` and the Claude handler
wiring, then let the normal GitHub Actions workflow rebuild and deploy the
previous behavior. Resin does not require a matching rollback because its
source, Account leases, retry budgets, and node configuration are unchanged.

During an application rollback, slow xAI first-token waits may again exceed
Cloudflare's origin response window. Existing Nginx timeouts do not prevent that
client-side 524 behavior because no downstream bytes are sent during bootstrap.

## Inspection timer

The deployment switch is independent of the image behavior:

- Keep `ENABLE_GROK_INSPECTION_TIMER=false` to preserve the current disabled
  state across any rollback deployment.
- Set it to `true` and rerun the production deploy only when the operator wants
  the five-minute inspection timer restored.

If the helper itself must be rolled back, disable the units explicitly after
deployment with `systemctl disable --now grok-inspection.timer` and
`systemctl stop grok-inspection.service`.
