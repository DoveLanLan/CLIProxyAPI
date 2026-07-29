#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=configure-grok-inspection-timer.sh
source "$SCRIPT_DIR/configure-grok-inspection-timer.sh"

TEST_TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/cliproxy-grok-timer.XXXXXX")"
trap 'rm -rf "$TEST_TMP_DIR"' EXIT

CALL_LOG="$TEST_TMP_DIR/calls.log"
SERVICE_SOURCE="$TEST_TMP_DIR/grok-inspection.service"
TIMER_SOURCE="$TEST_TMP_DIR/grok-inspection.timer"
SYSTEMD_DIR="$TEST_TMP_DIR/systemd"
mkdir -p "$SYSTEMD_DIR"
touch "$SERVICE_SOURCE" "$TIMER_SOURCE"

install() {
  printf 'install %s\n' "$*" >> "$CALL_LOG"
}

systemctl() {
  printf 'systemctl %s\n' "$*" >> "$CALL_LOG"
}

assert_log_equals() {
  local expected="$1"
  local actual
  actual="$(cat "$CALL_LOG")"
  if [[ "$actual" != "$expected" ]]; then
    echo "unexpected command log" >&2
    printf 'expected:\n%s\nactual:\n%s\n' "$expected" "$actual" >&2
    exit 1
  fi
}

: > "$CALL_LOG"
configure_grok_inspection_timer true "$SERVICE_SOURCE" "$TIMER_SOURCE" "$SYSTEMD_DIR"
assert_log_equals "$(printf '%s\n' \
  "install -m 644 $SERVICE_SOURCE $SYSTEMD_DIR/grok-inspection.service" \
  "install -m 644 $TIMER_SOURCE $SYSTEMD_DIR/grok-inspection.timer" \
  "systemctl daemon-reload" \
  "systemctl enable --now grok-inspection.timer")"

: > "$CALL_LOG"
configure_grok_inspection_timer false "$SERVICE_SOURCE" "$TIMER_SOURCE" "$SYSTEMD_DIR"
assert_log_equals "$(printf '%s\n' \
  "install -m 644 $SERVICE_SOURCE $SYSTEMD_DIR/grok-inspection.service" \
  "install -m 644 $TIMER_SOURCE $SYSTEMD_DIR/grok-inspection.timer" \
  "systemctl daemon-reload" \
  "systemctl disable --now grok-inspection.timer" \
  "systemctl stop grok-inspection.service")"

: > "$CALL_LOG"
if configure_grok_inspection_timer invalid "$SERVICE_SOURCE" "$TIMER_SOURCE" "$SYSTEMD_DIR" 2>/dev/null; then
  echo "invalid timer setting unexpectedly succeeded" >&2
  exit 1
fi
if [[ -s "$CALL_LOG" ]]; then
  echo "invalid timer setting invoked state-changing commands" >&2
  exit 1
fi

echo "Grok inspection timer configuration checks passed"
