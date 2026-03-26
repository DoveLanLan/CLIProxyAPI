#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMPOSE_FILE="$ROOT_DIR/compose.production.yml"

mkdir -p "$ROOT_DIR/data/auths" "$ROOT_DIR/data/logs" "$ROOT_DIR/certs"

if [[ ! -f "$ROOT_DIR/.env" && -f "$ROOT_DIR/.env.example" ]]; then
  cp "$ROOT_DIR/.env.example" "$ROOT_DIR/.env"
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
docker compose -f "$COMPOSE_FILE" pull
docker compose -f "$COMPOSE_FILE" up -d --remove-orphans
docker compose -f "$COMPOSE_FILE" ps
