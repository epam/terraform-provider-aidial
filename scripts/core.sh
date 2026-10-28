#!/usr/bin/env bash
# Start or stop the throwaway DIAL Core defined in tests/core/docker-compose.yml.
#
#   scripts/core.sh up     start Core + Valkey and wait until /health answers
#   scripts/core.sh down   remove containers, networks and volumes
#   scripts/core.sh url    print the Core base URL
#
# External Core: with CORE_URL set, or when no Docker daemon is reachable (the devcontainer),
# `up` only waits for that Core's /health, `down` does nothing and Docker is not needed.
# CORE_URL defaults to http://host.docker.internal:CORE_PORT (or the container's default
# gateway when that name does not resolve), i.e. a Core started by this script on the host.
# It never removes an external Core. Set CORE_URL=http://localhost:8080 to use a Core on
# this machine while Docker is available.
#
# Env: CORE_URL (external Core, see above), CORE_VERSION (image tag, default latest), CORE_HOST (default localhost),
#      CORE_PORT (default 8080), CORE_BIND (publish address, default 0.0.0.0 so
#      containers can reach it; use 127.0.0.1 to keep it local), CORE_WAIT (seconds, default 120).
#      DOCKER_HOST selects the Docker daemon.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export CORE_VERSION="${CORE_VERSION:-latest}"
export CORE_PORT="${CORE_PORT:-8080}"
export CORE_BIND="${CORE_BIND:-0.0.0.0}"
host="${CORE_HOST:-localhost}"
compose=(docker compose -f "$root/tests/core/docker-compose.yml")

# Host to reach a Core running on the Docker host from inside a container.
container_host_gateway() {
  if getent hosts host.docker.internal >/dev/null 2>&1; then
    echo host.docker.internal
    return
  fi
  # Not mapped (devcontainer not rebuilt yet): fall back to the default route's gateway.
  hex="$(awk '$2=="00000000" {print $3; exit}' /proc/net/route 2>/dev/null || true)"
  if [ -n "$hex" ]; then
    printf '%d.%d.%d.%d\n' "0x${hex:6:2}" "0x${hex:4:2}" "0x${hex:2:2}" "0x${hex:0:2}"
  else
    echo host.docker.internal
  fi
}

external="${CORE_URL:-}"
if [ -z "$external" ] && ! docker info >/dev/null 2>&1; then
  external="http://$(container_host_gateway):${CORE_PORT}"
fi
url="${external:-http://${host}:${CORE_PORT}}"
url="${url%/}"

need_docker() {
  if ! docker info >/dev/null 2>&1; then
    echo "No reachable Docker daemon. Set DOCKER_HOST (e.g. tcp://docker:2375 for a docker:dind sidecar)" >&2
    echo "in ~/devcontainer.env, or run on a host with Docker. See README \"Testing against a local Core\"." >&2
    exit 1
  fi
}

wait_healthy() {
  deadline=$((SECONDS + ${CORE_WAIT:-120}))
  # Any HTTP answer counts as up: newer Core versions answer /health with 401 without a key.
  until [ "$(curl -s -o /dev/null -w '%{http_code}' "$url/health" 2>/dev/null)" != "000" ]; do
    if [ "$SECONDS" -ge "$deadline" ]; then
      echo "DIAL Core did not become healthy at $url" >&2
      return 1
    fi
    sleep 2
  done
}

case "${1:-}" in
  up)
    if [ -n "$external" ]; then
      wait_healthy || { echo "Start it on the host with: scripts/core.sh up" >&2; exit 1; }
      echo "Using external DIAL Core at $url"
      exit 0
    fi
    need_docker
    "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
    if ! "${compose[@]}" up -d --wait; then
      "${compose[@]}" logs --tail 100 >&2 || true
      "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
      exit 1
    fi
    if ! wait_healthy; then
      "${compose[@]}" logs --tail 100 core >&2 || true
      "${compose[@]}" down -v --remove-orphans >/dev/null 2>&1 || true
      exit 1
    fi
    echo "DIAL Core ${CORE_VERSION} is up at $url"
    ;;
  down)
    if [ -n "$external" ]; then
      echo "External DIAL Core at $url left running"
      exit 0
    fi
    need_docker
    "${compose[@]}" down -v --remove-orphans
    ;;
  url)
    echo "$url"
    ;;
  *)
    echo "usage: $0 up|down|url" >&2
    exit 2
    ;;
esac
