# Rollback Notes: xAI streamed free-usage failover and Grok account-pool repair

- Date: 2026-07-23
- Related: `spec.md`, `tasks.md`

## Code and deployment rollback

- Revert `3834ed4d` to remove streamed xAI error classification and its tests.
- Revert `e013d98c` to restore the prior tracked inspection runner behavior.
- Redeploy the previous image revision `864edbdb98da455734c735d81e8bf5f23fa04420` only if the new executor causes a confirmed regression.

## Configuration rollback

- Restore `/opt/cliproxyapi/backups/grok-remediation-20260723T121222+0800/config.yaml.pre-routing-20260723T060933+0000` through the validated management config endpoint.
- That backup restores `routing.strategy: fill-first` and removes the explicit cooldown-persistence settings added during this change.
- A container restart is not required when the management config hot reload succeeds; verify the management API after restoration.

## Credential rollback

- Initial deleted credentials are archived in `/opt/cliproxyapi/backups/grok-remediation-20260723T121222+0800/delete-auth-files.tgz` (479 files).
- Follow-up deleted credentials are archived in `/opt/cliproxyapi/backups/grok-remediation-20260723T121222+0800/followup-reauth-20260723T070750+0000/auth-files.tgz` (2 files).
- Verify each archive against its SHA-256 manifest before restoring through the management auth upload API.
- Restored reauth credentials remain invalid until re-login; restoring them is for data recovery, not for returning them to active rotation.

## Operational caution

Rolling back the executor restores the original Claude Code failure shape: an upstream HTTP 200 SSE quota error may reach the client instead of triggering transparent credential failover. Prefer targeted rollback only after a confirmed regression.

## 2026-07-24 systemd policy rollback

- Revert the commit that adds `deploy/systemd/grok-inspection.service`, `deploy/systemd/grok-inspection.timer`, and the install block in `deploy/scripts/remote-deploy.sh`.
- On production, restore `/opt/cliproxyapi/backups/grok-remediation-20260723T121222+0800/grok-inspection.service.pre-permission-denied-20260724T012714+0000`, then run `systemctl daemon-reload`.
- Restoring the old unit re-enables the defect: `permission_denied` credentials remain eligible for routing. Prefer disabling the timer instead of restoring that policy unless rollback is required for an unrelated unit failure.
- The 112 credentials disabled by the follow-up were not deleted and require no data restore; they can be re-enabled through the management API if the classification policy is intentionally reversed.
