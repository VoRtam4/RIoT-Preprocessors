#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PROJECT_ROOT="$SCRIPT_DIR/../RIoT"
PREPROCESSORS_ROOT="$SCRIPT_DIR"
MAIN_PROJECT="$PROJECT_ROOT"

COMMONS_SRC="$PROJECT_ROOT/backend/commons"
COMMONS_DST="$PREPROCESSORS_ROOT/backend/commons"

DOCKER_DIR="$PROJECT_ROOT/docker"

sync_commons() {
  if [ ! -d "$COMMONS_SRC" ]; then
    echo "SOURCE neexistuje: $COMMONS_SRC"
    exit 1
  fi

  mkdir -p "$COMMONS_DST"

  rsync -av \
    "$COMMONS_SRC/" \
    "$COMMONS_DST/"
}

clean_main_docker() {
  if [ -d "$DOCKER_DIR" ]; then
    echo "Mažu $DOCKER_DIR"
    rm -rf "$DOCKER_DIR"
  fi
}

stop_all() {
  cd "$PREPROCESSORS_ROOT"
  docker-compose down -v || true

  cd "$MAIN_PROJECT"
  docker-compose down -v || true
}

start_all() {
  cd "$MAIN_PROJECT"
  docker-compose up --build -d

  cd "$PREPROCESSORS_ROOT"
  docker-compose up --build -d
}

restart_all() {
  stop_all
  sync_commons
  clean_main_docker
  start_all
}

case "${1:-}" in
  stop)
    stop_all
    ;;
  restart)
    restart_all
    ;;
  *)
    echo "Usage: ./run.sh [stop|restart]"
    ;;
esac