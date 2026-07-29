#!/usr/bin/env bash

validate_grok_inspection_timer_setting() {
  local enabled="${1:-true}"
  case "$enabled" in
    true|false)
      return
      ;;
    *)
      echo "error: ENABLE_GROK_INSPECTION_TIMER must be true or false" >&2
      return 1
      ;;
  esac
}

configure_grok_inspection_timer() {
  local enabled="${1:-true}"
  local service_source="${2:?Grok inspection service source is required}"
  local timer_source="${3:?Grok inspection timer source is required}"
  local systemd_dir="${4:-/etc/systemd/system}"

  if ! validate_grok_inspection_timer_setting "$enabled"; then
    return 1
  fi

  install -m 644 "$service_source" "$systemd_dir/grok-inspection.service"
  install -m 644 "$timer_source" "$systemd_dir/grok-inspection.timer"
  systemctl daemon-reload

  if [[ "$enabled" == "true" ]]; then
    systemctl enable --now grok-inspection.timer
    return
  fi

  systemctl disable --now grok-inspection.timer
  systemctl stop grok-inspection.service
}
