#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PROJECT_ROOT="$SCRIPT_DIR/../RIoT" # <-- adjust if your project is in a different location
PREPROCESSORS_ROOT="$SCRIPT_DIR"
MAIN_PROJECT="$PROJECT_ROOT"

DOCKER_DIR="$PROJECT_ROOT/docker"

sync_backend() {
  mkdir -p "$PREPROCESSORS_ROOT/backend"

  for dir in \
    WazeJam_preprocessor \
    datex-downloader \
    mhd-preprocessor \
    ndic-preprocessor \
    commons
  do
    echo "Sync $dir..."
    rsync -av --delete \
      --exclude 'targetApp/' \
      --exclude 'targetAppFE/' \
      --exclude 'targetAppBE/' \
      --exclude 'node_modules/' \
      --exclude '*.log' \
      "$PROJECT_ROOT/backend/$dir/" \
      "$PREPROCESSORS_ROOT/backend/$dir/"
  done
}

clean_main_docker() {
  if [ -d "$DOCKER_DIR" ]; then
    echo "Mažu $DOCKER_DIR"
    rm -rf "$DOCKER_DIR"
  fi
}

stop_all() {
  cd "$PREPROCESSORS_ROOT"
  docker-compose down || true

  cd "$MAIN_PROJECT"
  docker-compose down || true
}

start_all() {
  cd "$MAIN_PROJECT"
  docker-compose up --build -d

  cd "$PREPROCESSORS_ROOT"
  docker-compose build --no-cache
  docker-compose up -d
}

restart_all() {
  stop_all
  sync_backend
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
    echo "Usage: ./make.sh [stop|restart]"
    ;;
esac