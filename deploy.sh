#!/usr/bin/env bash
set -euo pipefail

cd "$HOME/max"

COMPOSE=(docker compose -f docker-compose.prod.yml)

"${COMPOSE[@]}" pull postgres backend frontend
"${COMPOSE[@]}" up -d --remove-orphans postgres backend frontend

docker image prune -f
