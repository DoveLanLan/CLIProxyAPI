# Rollback

Revert this task's executor/helper/test changes and deploy the previous immutable
image sha-eb9239145b29cb4e9356b79959643588c84f4a31. No config or data migration.
This restores the known tool-image 400, so prefer forward fixes if possible.
Do not restore the earlier blanket DeepSeek text-only preflight.
