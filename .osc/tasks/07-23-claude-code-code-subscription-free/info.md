# Tech notes

## Architecture decisions

- Detect xAI HTTP SSE error objects before protocol translation so the shared auth manager owns cooldown and failover.
- Keep rolling quota and transient probe failures recoverable; never treat disabled state alone as deletion evidence.
- Recover disabled accounts only after a fresh `healthy` result and explicit safe-recover mode.

## Production evidence

- Inspection result checksum: `ec961346a514522858c08c91aba405a9b1c2b7c3b6775d09e5d865761ffaa5bb`.
- Final disabled-only classification: healthy 1346, quota 55, probe error 92, permission denied 1023, reauth 479.
- Deleted: 479 fresh HTTP-401 reauth results with plugin `action=delete`.
- Kept: quota, probe errors, HTTP-402 spending-limit permission failures, and other ambiguous states.

## Risks / mitigations

- Production code is not active until the main image is deployed; do not re-enable the 1346 healthy credentials before that point.
- A restricted backup contains all 479 deleted auth files plus the immutable inspection result and apply manifests.

## Rollback plan

- Restore deleted auth files from the restricted production backup only if the classification or delete batch is later found incorrect.
- Revert commits `e013d98c` and `3834ed4d` to restore previous code/script behavior.
