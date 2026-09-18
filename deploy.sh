#!/usr/bin/env bash
set -euo pipefail

cd "$HOME/max"

COMPOSE=(docker compose -f docker-compose.prod.yml)

"${COMPOSE[@]}" pull postgres gar-init backend frontend

if [[ -f .demo-reset-maintenance ]]; then
  echo "Demo database reset maintenance is active; application services remain stopped"
  "${COMPOSE[@]}" stop frontend backend gar-init
  exit 0
fi

"${COMPOSE[@]}" up -d --remove-orphans postgres gar-init backend frontend

docker image prune -f
