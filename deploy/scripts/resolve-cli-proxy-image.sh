#!/usr/bin/env bash

resolve_cli_proxy_image() {
  local env_file="$1"
  local requested_cli_proxy_image="${CLI_PROXY_IMAGE:-}"

  if [[ -f "$env_file" ]]; then
    # shellcheck disable=SC1090
    source "$env_file"
  fi

  if [[ -n "$requested_cli_proxy_image" ]]; then
    CLI_PROXY_IMAGE="$requested_cli_proxy_image"
  fi
  export CLI_PROXY_IMAGE
}
