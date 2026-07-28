#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=resolve-cli-proxy-image.sh
source "$SCRIPT_DIR/resolve-cli-proxy-image.sh"

TEST_TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/cliproxy-image-env.XXXXXX")"
trap 'rm -rf "$TEST_TMP_DIR"' EXIT

ENV_FILE="$TEST_TMP_DIR/.env"
printf '%s\n' 'CLI_PROXY_IMAGE=ghcr.io/example/cliproxyapi:from-env' > "$ENV_FILE"

explicit_image="$(
  CLI_PROXY_IMAGE='ghcr.io/example/cliproxyapi:sha-workflow'
  resolve_cli_proxy_image "$ENV_FILE"
  printf '%s' "$CLI_PROXY_IMAGE"
)"
if [[ "$explicit_image" != 'ghcr.io/example/cliproxyapi:sha-workflow' ]]; then
  echo "explicit CLI_PROXY_IMAGE did not take precedence" >&2
  exit 1
fi

fallback_image="$(
  unset CLI_PROXY_IMAGE
  resolve_cli_proxy_image "$ENV_FILE"
  printf '%s' "$CLI_PROXY_IMAGE"
)"
if [[ "$fallback_image" != 'ghcr.io/example/cliproxyapi:from-env' ]]; then
  echo ".env CLI_PROXY_IMAGE was not used as the fallback" >&2
  exit 1
fi

echo "CLI_PROXY_IMAGE precedence checks passed"
