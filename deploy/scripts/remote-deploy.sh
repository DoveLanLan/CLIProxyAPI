#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/compose.production.yml"
COMPOSE_ARGS=(-f "$COMPOSE_FILE")

mkdir -p "$ROOT_DIR/data/auths" "$ROOT_DIR/data/logs" "$ROOT_DIR/certs"

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

if [[ ! -f "$ROOT_DIR/data/config.yaml" ]]; then
  echo "error: missing $ROOT_DIR/data/config.yaml" >&2
  echo "Place your runtime config on the server before deploying." >&2
  exit 1
fi

if [[ ! -f "$ROOT_DIR/certs/origin.crt" ]]; then
  echo "error: missing $ROOT_DIR/certs/origin.crt" >&2
  exit 1
fi

if [[ ! -f "$ROOT_DIR/certs/origin.key" ]]; then
  echo "error: missing $ROOT_DIR/certs/origin.key" >&2
  exit 1
fi

: "${CLI_PROXY_IMAGE:?CLI_PROXY_IMAGE must be set}"

cd "$ROOT_DIR"
docker compose "${COMPOSE_ARGS[@]}" pull
docker compose "${COMPOSE_ARGS[@]}" up -d --remove-orphans
docker compose "${COMPOSE_ARGS[@]}" ps
