#!/usr/bin/env bash
# Auto-trigger CLIProxyAPI grok-inspection (start + wait + safe apply).
#
# Safe apply (default ON):
#   Only force_action=disable for permanent authentication failures.
#   Rolling quota exhaustion and transient probe errors remain recoverable.
#   Never delete. Never enable.
#   Skips accounts already disabled so re-runs stay cheap on small VPS.
#
# Env:
#   CPA_MGMT_BASE   default http://100.67.99.9:18318
#   CPA_MANAGEMENT_KEY / CPA_MGMT_KEY_FILE
#   GROK_INSPECT_MODE   incremental|full|filter  default incremental
#   GROK_INSPECT_WORKERS  1-16  default 3  (keep low on 2c/2G boxes)
#   GROK_INSPECT_INCLUDE_DISABLED  0|1  default 0
#   GROK_INSPECT_ONLY_DISABLED     0|1  default 0
#   GROK_INSPECT_MAX_WAIT_SEC      default 1800
#   GROK_INSPECT_POLL_SEC          default 10
#   GROK_INSPECT_SAFE_APPLY        1|0  default 1
#   GROK_INSPECT_DISABLE_CLASSES   comma list of permanent failure classes
#   GROK_INSPECT_RESULTS_JSON      optional local results path for cheap filter
set -euo pipefail

BASE="${CPA_MGMT_BASE:-http://100.67.99.9:18318}"
KEY_FILE="${CPA_MGMT_KEY_FILE:-/opt/cliproxyapi/data/.management-key}"
MODE="${GROK_INSPECT_MODE:-incremental}"
WORKERS="${GROK_INSPECT_WORKERS:-3}"
INCLUDE_DISABLED="${GROK_INSPECT_INCLUDE_DISABLED:-0}"
ONLY_DISABLED="${GROK_INSPECT_ONLY_DISABLED:-0}"
MAX_WAIT="${GROK_INSPECT_MAX_WAIT_SEC:-1800}"
POLL="${GROK_INSPECT_POLL_SEC:-10}"
SAFE_APPLY="${GROK_INSPECT_SAFE_APPLY:-1}"
DISABLE_CLASSES="${GROK_INSPECT_DISABLE_CLASSES:-invalid_grant,reauth,deactivated,banned,permission_denied}"
RESULTS_JSON="${GROK_INSPECT_RESULTS_JSON:-/opt/cliproxyapi/data/grok-inspection/results.json}"
LOG_TAG="[grok-inspection-cron]"

log() { printf '%s %s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$LOG_TAG" "$*"; }

if [[ -z "${CPA_MANAGEMENT_KEY:-}" ]]; then
  if [[ -r "$KEY_FILE" ]]; then
    CPA_MANAGEMENT_KEY="$(tr -d '\r\n' <"$KEY_FILE")"
  fi
fi
if [[ -z "${CPA_MANAGEMENT_KEY:-}" ]]; then
  log "ERROR: set CPA_MANAGEMENT_KEY or put the key in $KEY_FILE (chmod 600)"
  exit 1
fi

if ! [[ "$WORKERS" =~ ^[0-9]+$ ]] || ((WORKERS < 1 || WORKERS > 16)); then
  log "WARN: invalid workers=$WORKERS, fallback 3"
  WORKERS=3
fi

include_json=false
only_json=false
[[ "$INCLUDE_DISABLED" == "1" || "$INCLUDE_DISABLED" == "true" ]] && include_json=true
[[ "$ONLY_DISABLED" == "1" || "$ONLY_DISABLED" == "true" ]] && only_json=true

case "$MODE" in
incremental | incr)
  BODY=$(printf '{"workers":%s,"include_disabled":%s,"only_disabled":%s,"incremental":true}' \
    "$WORKERS" "$include_json" "$only_json")
  ;;
full)
  BODY=$(printf '{"workers":%s,"include_disabled":%s,"only_disabled":%s,"incremental":false}' \
    "$WORKERS" "$include_json" "$only_json")
  ;;
filter)
  BODY=$(printf '{"workers":%s,"include_disabled":%s,"only_disabled":%s,"incremental":false,"mode":"filter"}' \
    "$WORKERS" "$include_json" "$only_json")
  ;;
*)
  log "ERROR: unknown GROK_INSPECT_MODE=$MODE (use incremental|full|filter)"
  exit 2
  ;;
esac

auth_hdr=(-H "Authorization: Bearer ${CPA_MANAGEMENT_KEY}" -H "X-Management-Key: ${CPA_MANAGEMENT_KEY}")
json_hdr=(-H "Content-Type: application/json" -H "Accept: application/json")
curl_common=(curl -fsS --connect-timeout 10 --max-time 120)

api_get() {
  local path="$1"
  "${curl_common[@]}" "${auth_hdr[@]}" "${BASE}${path}"
}

api_post() {
  local path="$1"
  local body="$2"
  "${curl_common[@]}" -X POST "${auth_hdr[@]}" "${json_hdr[@]}" -d "$body" "${BASE}${path}"
}

status_json() {
  api_get "/v0/management/plugins/grok-inspection/status?include_results=0"
}

# Skip if already running/applying (prevents pile-up when interval < run time).
if st=$(status_json 2>/dev/null); then
  if printf '%s' "$st" | grep -qiE '"running"[[:space:]]*:[[:space:]]*true'; then
    log "skip: inspection already running"
    exit 0
  fi
  if printf '%s' "$st" | grep -qiE '"applying"[[:space:]]*:[[:space:]]*true'; then
    log "skip: apply already in progress"
    exit 0
  fi
else
  log "WARN: status probe failed (will still try start): $BASE"
fi

log "start mode=$MODE workers=$WORKERS safe_apply=$SAFE_APPLY base=$BASE"
start_resp="$(api_post "/v0/management/plugins/grok-inspection/start" "$BODY" 2>/dev/null || true)"
if [[ -n "$start_resp" ]]; then
  # Avoid a broken pipe from head on large JSON.
  log "start response: $(printf '%s' "$start_resp" | python3 -c 'import sys; print(sys.stdin.read()[:300].replace("\n"," "))' 2>/dev/null || echo ok)"
  if printf '%s' "$start_resp" | grep -qiE 'already running|busy'; then
    log "skip: server reports already running"
    exit 0
  fi
else
  if ! status_json >/dev/null 2>&1; then
    log "ERROR: start request failed and management API unreachable"
    exit 3
  fi
  log "start posted (empty body)"
fi

deadline=$(($(date +%s) + MAX_WAIT))
finished=0
while (($(date +%s) < deadline)); do
  if ! st=$(status_json 2>/dev/null); then
    log "WARN: status poll failed, retry in ${POLL}s"
    sleep "$POLL"
    continue
  fi
  running=$(printf '%s' "$st" | grep -oE '"running"[[:space:]]*:[[:space:]]*(true|false)' | head -1 | grep -oE 'true|false' || true)
  applying=$(printf '%s' "$st" | grep -oE '"applying"[[:space:]]*:[[:space:]]*(true|false)' | head -1 | grep -oE 'true|false' || true)
  done_n=$(printf '%s' "$st" | grep -oE '"done"[[:space:]]*:[[:space:]]*[0-9]+' | head -1 | grep -oE '[0-9]+$' || true)
  total_n=$(printf '%s' "$st" | grep -oE '"total"[[:space:]]*:[[:space:]]*[0-9]+' | head -1 | grep -oE '[0-9]+$' || true)
  log "status running=${running:-?} applying=${applying:-?} done=${done_n:-?}/${total_n:-?}"

  if [[ "${running:-}" == "false" || -z "${running:-}" ]]; then
    finished=1
    break
  fi
  sleep "$POLL"
done

if ((finished == 0)); then
  log "ERROR: timed out after ${MAX_WAIT}s waiting for inspection to finish"
  exit 4
fi
log "inspection finished"

if [[ "$SAFE_APPLY" != "1" && "$SAFE_APPLY" != "true" ]]; then
  log "safe apply disabled; done"
  exit 0
fi

# Build a list of auth indexes that still need disabling. Rolling quota and
# transient failures are intentionally absent from the permanent class list.
# Prefer local results.json (cheap); fall back to status?include_results=1.
APPLY_BODY=$(
  DISABLE_CLASSES="$DISABLE_CLASSES" RESULTS_JSON="$RESULTS_JSON" BASE="$BASE" \
    CPA_MANAGEMENT_KEY="$CPA_MANAGEMENT_KEY" python3 - <<'PY'
import json, os, urllib.request

classes = {x.strip().lower() for x in os.environ.get("DISABLE_CLASSES", "").split(",") if x.strip()}
classes -= {"healthy", "delete", "enable", "quota_exhausted", "probe_error"}
path = os.environ.get("RESULTS_JSON", "")
rows = []

def load_rows_from_obj(data):
    if isinstance(data, list):
        return data
    if isinstance(data, dict):
        for key in ("results", "rows", "items"):
            if isinstance(data.get(key), list):
                return data[key]
    return []

if path and os.path.isfile(path):
    try:
        with open(path, "r", encoding="utf-8") as results_file:
            rows = load_rows_from_obj(json.load(results_file))
    except Exception:
        rows = []

if not rows:
    base = os.environ["BASE"].rstrip("/")
    key = os.environ["CPA_MANAGEMENT_KEY"]
    request = urllib.request.Request(
        base + "/v0/management/plugins/grok-inspection/status?include_results=1",
        headers={
            "X-Management-Key": key,
            "Authorization": "Bearer " + key,
            "Accept": "application/json",
        },
        method="GET",
    )
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            rows = load_rows_from_obj(json.loads(response.read().decode("utf-8", "replace")))
    except Exception:
        rows = []

indexes = []
for row in rows:
    if not isinstance(row, dict) or row.get("disabled") is True:
        continue
    classification = str(row.get("classification") or row.get("class") or "").strip().lower()
    if classification not in classes:
        continue
    index = row.get("auth_index") or row.get("file_name") or row.get("name") or ""
    index = str(index).strip()
    if index:
        indexes.append(index)

seen = set()
unique_indexes = []
for index in indexes:
    if index in seen:
        continue
    seen.add(index)
    unique_indexes.append(index)

print(json.dumps({"count": len(unique_indexes), "body": {"force_action": "disable", "auth_indexes": unique_indexes}, "classes": sorted(classes)}))
PY
)

COUNT=$(printf '%s' "$APPLY_BODY" | python3 -c 'import sys,json; print(json.load(sys.stdin).get("count",0))')
CLASSES_USED=$(printf '%s' "$APPLY_BODY" | python3 -c 'import sys,json; print(",".join(json.load(sys.stdin).get("classes",[])))')
if [[ "${COUNT:-0}" -eq 0 ]]; then
  log "safe apply skip: 0 accounts need disable (classes=$CLASSES_USED)"
  exit 0
fi

BODY_JSON=$(printf '%s' "$APPLY_BODY" | python3 -c 'import sys,json; print(json.dumps(json.load(sys.stdin)["body"]))')
log "safe apply: force_action=disable targets=$COUNT classes=$CLASSES_USED"
apply_resp="$(api_post "/v0/management/plugins/grok-inspection/apply" "$BODY_JSON" 2>/dev/null || true)"
if [[ -n "$apply_resp" ]]; then
  log "apply response: $(printf '%s' "$apply_resp" | python3 -c 'import sys; print(sys.stdin.read()[:400])' 2>/dev/null || true)"
else
  log "apply posted (empty body)"
fi

deadline=$(($(date +%s) + MAX_WAIT))
while (($(date +%s) < deadline)); do
  if ! st=$(status_json 2>/dev/null); then
    sleep "$POLL"
    continue
  fi
  applying=$(printf '%s' "$st" | grep -oE '"applying"[[:space:]]*:[[:space:]]*(true|false)' | head -1 | grep -oE 'true|false' || true)
  adone=$(printf '%s' "$st" | grep -oE '"apply_done"[[:space:]]*:[[:space:]]*[0-9]+' | head -1 | grep -oE '[0-9]+$' || true)
  atotal=$(printf '%s' "$st" | grep -oE '"apply_total"[[:space:]]*:[[:space:]]*[0-9]+' | head -1 | grep -oE '[0-9]+$' || true)
  log "apply status applying=${applying:-?} ${adone:-?}/${atotal:-?}"
  if [[ "${applying:-}" != "true" ]]; then
    log "apply finished"
    exit 0
  fi
  sleep "$POLL"
done

log "ERROR: timed out waiting for apply"
exit 5
