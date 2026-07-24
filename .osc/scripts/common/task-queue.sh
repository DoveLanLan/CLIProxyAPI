#!/usr/bin/env bash
set -euo pipefail

_COMMON_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$_COMMON_DIR/paths.sh"

osc_list_task_dirs() {
  local root="${1:-$(osc_repo_root)}"
  local tasks
  tasks="$(osc_tasks_dir "$root")"
  [[ -d "$tasks" ]] || return 0
  find "$tasks" -maxdepth 1 -mindepth 1 -type d -print | sort
}

osc_archive_dir() {
  local root="${1:-$(osc_repo_root)}"
  echo "$(osc_tasks_dir "$root")/$DIR_ARCHIVE/$(date +%Y-%m)"
}

osc_find_next_task_rel() {
  local root="${1:-$(osc_repo_root)}"

  while IFS= read -r d; do
    local tj="$d/$FILE_TASK_JSON"
    [[ -f "$tj" ]] || continue

    local status
    status="$(jq -r '.status // "planned"' "$tj" 2>/dev/null)"
    [[ "$status" == "done" || "$status" == "completed" || "$status" == "archived" ]] && continue

    local blocked=false
    local deps
    deps="$(jq -r '.depends_on // [] | .[]' "$tj" 2>/dev/null)"
    while IFS= read -r dep; do
      [[ -n "$dep" ]] || continue
      local dep_json="$root/$dep/$FILE_TASK_JSON"
      if [[ -f "$dep_json" ]]; then
        local dep_status
        dep_status="$(jq -r '.status // "planned"' "$dep_json" 2>/dev/null)"
        if [[ "$dep_status" != "done" && "$dep_status" != "completed" && "$dep_status" != "archived" ]]; then
          blocked=true
          break
        fi
      fi
    done <<< "$deps"
    [[ "$blocked" == "true" ]] && continue

    echo "${d#$root/}"
    return 0
  done < <(osc_list_task_dirs "$root")

  return 0
}
