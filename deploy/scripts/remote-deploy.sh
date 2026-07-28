#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/compose.production.yml"
COMPOSE_ARGS=(-f "$COMPOSE_FILE")
NGINX_SOURCE="$ROOT_DIR/nginx/conf.d/api.heweili.top.conf"
GROK_INSPECTION_SERVICE_SOURCE="$ROOT_DIR/systemd/grok-inspection.service"
GROK_INSPECTION_TIMER_SOURCE="$ROOT_DIR/systemd/grok-inspection.timer"

mkdir -p \
  "$ROOT_DIR/data/auths" \
  "$ROOT_DIR/data/logs" \
  "$ROOT_DIR/data/plugins" \
  "$ROOT_DIR/data/grok-inspection"

if [[ ! -f "$ROOT_DIR/.env" && -f "$ROOT_DIR/.env.example" ]]; then
  cp "$ROOT_DIR/.env.example" "$ROOT_DIR/.env"
fi

if [[ -f "$ROOT_DIR/.env" ]]; then
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env"
fi

if [[ "${ENABLE_SPLIT_PROXY:-false}" == "true" ]]; then
  COMPOSE_ARGS+=(-f "$ROOT_DIR/compose.production.split-proxy.yml")
fi

if [[ "${ENABLE_XAI_PROXY_POOL:-false}" == "true" && "${ENABLE_XAI_RESIN_PROXY:-false}" == "true" ]]; then
  echo "error: ENABLE_XAI_PROXY_POOL and ENABLE_XAI_RESIN_PROXY are mutually exclusive" >&2
  exit 1
fi

if [[ "${ENABLE_XAI_PROXY_POOL:-false}" == "true" ]]; then
  COMPOSE_ARGS+=(-f "$ROOT_DIR/compose.production.xai-proxy.yml")
  EGRESS_PROXY_NETWORK="${EGRESS_PROXY_NETWORK:-egress-proxy}"
  EGRESS_PROXY_API_TOKEN="${EGRESS_PROXY_API_TOKEN:-/opt/egress-proxy-pool/secrets/api-token}"
  if [[ ! -s "$EGRESS_PROXY_API_TOKEN" ]]; then
    echo "error: missing EgressProxyPool API token file: $EGRESS_PROXY_API_TOKEN" >&2
    exit 1
  fi
  if ! docker network inspect "$EGRESS_PROXY_NETWORK" >/dev/null 2>&1; then
    echo "error: missing EgressProxyPool Docker network: $EGRESS_PROXY_NETWORK" >&2
    echo "Start the standalone EgressProxyPool Compose project before enabling this overlay." >&2
    exit 1
  fi
  chmod 600 "$EGRESS_PROXY_API_TOKEN"
  export EGRESS_PROXY_NETWORK EGRESS_PROXY_API_TOKEN
fi

if [[ "${ENABLE_XAI_RESIN_PROXY:-false}" == "true" ]]; then
  COMPOSE_ARGS+=(-f "$ROOT_DIR/compose.production.xai-resin.yml")
  XAI_RESIN_PROXY_TOKEN_FILE="${XAI_RESIN_PROXY_TOKEN_FILE:-/opt/resin/secrets/proxy-token}"
  XAI_RESIN_IDENTITY_KEY_FILE="${XAI_RESIN_IDENTITY_KEY_FILE:-/opt/cliproxyapi/secrets/resin-identity-key}"
  if [[ ! -s "$XAI_RESIN_PROXY_TOKEN_FILE" ]]; then
    echo "error: missing Resin proxy token file: $XAI_RESIN_PROXY_TOKEN_FILE" >&2
    exit 1
  fi
  if [[ ! -s "$XAI_RESIN_IDENTITY_KEY_FILE" ]]; then
    echo "error: missing Resin identity key file: $XAI_RESIN_IDENTITY_KEY_FILE" >&2
    exit 1
  fi
  chmod 600 "$XAI_RESIN_PROXY_TOKEN_FILE" "$XAI_RESIN_IDENTITY_KEY_FILE"
  export XAI_RESIN_PROXY_TOKEN_FILE XAI_RESIN_IDENTITY_KEY_FILE
fi

if [[ ! -f "$ROOT_DIR/data/config.yaml" ]]; then
  echo "error: missing $ROOT_DIR/data/config.yaml" >&2
  echo "Place your runtime config on the server before deploying." >&2
  exit 1
fi

: "${CLI_PROXY_IMAGE:?CLI_PROXY_IMAGE must be set}"

GATEWAY_NETWORK="${GATEWAY_NETWORK:-vps-gateway}"
GATEWAY_ROOT="${GATEWAY_ROOT:-/opt/vps-gateway}"
GATEWAY_CONTAINER="${GATEWAY_CONTAINER:-vps-gateway-nginx}"
LOCAL_CLAUDE_NETWORK="${LOCAL_CLAUDE_NETWORK:-cli-proxy-api-proxy}"
GATEWAY_CONF_DIR="$GATEWAY_ROOT/nginx/conf.d"

export GATEWAY_NETWORK GATEWAY_ROOT GATEWAY_CONTAINER LOCAL_CLAUDE_NETWORK

docker network inspect "$GATEWAY_NETWORK" >/dev/null 2>&1 || docker network create "$GATEWAY_NETWORK" >/dev/null

if [[ "${ENABLE_SPLIT_PROXY:-false}" == "true" ]]; then
  if ! docker network inspect "$LOCAL_CLAUDE_NETWORK" >/dev/null 2>&1; then
    echo "error: missing local Claude Docker network: $LOCAL_CLAUDE_NETWORK" >&2
    echo "Create it or set LOCAL_CLAUDE_NETWORK to the network shared with your local Claude-compatible service." >&2
    exit 1
  fi
fi

cd "$ROOT_DIR"
case "$CLI_PROXY_IMAGE" in
  cliproxyapi:*)
    if ! docker image inspect "$CLI_PROXY_IMAGE" >/dev/null 2>&1; then
      echo "error: missing local CLIProxyAPI image: $CLI_PROXY_IMAGE" >&2
      exit 1
    fi
    ;;
  *)
    docker compose "${COMPOSE_ARGS[@]}" pull
    ;;
esac
docker compose "${COMPOSE_ARGS[@]}" up -d --remove-orphans
docker compose "${COMPOSE_ARGS[@]}" ps

if command -v systemctl >/dev/null 2>&1 && [[ -d /run/systemd/system ]]; then
  install -m 644 "$GROK_INSPECTION_SERVICE_SOURCE" /etc/systemd/system/grok-inspection.service
  install -m 644 "$GROK_INSPECTION_TIMER_SOURCE" /etc/systemd/system/grok-inspection.timer
  systemctl daemon-reload
  systemctl enable --now grok-inspection.timer
else
  echo "warning: systemctl unavailable; Grok inspection timer was not installed" >&2
fi

if [[ ! -d "$GATEWAY_CONF_DIR" ]]; then
  echo "error: missing gateway config directory: $GATEWAY_CONF_DIR" >&2
  echo "Create the shared gateway stack before deploying CLIProxyAPI." >&2
  exit 1
fi

install -m 644 "$NGINX_SOURCE" "$GATEWAY_CONF_DIR/api.heweili.top.conf"

if ! docker ps --format '{{.Names}}' | grep -qx "$GATEWAY_CONTAINER"; then
  echo "error: gateway container is not running: $GATEWAY_CONTAINER" >&2
  exit 1
fi

docker exec "$GATEWAY_CONTAINER" nginx -t
docker exec "$GATEWAY_CONTAINER" nginx -s reload
