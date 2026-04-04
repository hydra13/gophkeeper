#!/bin/bash
set -euo pipefail

MODE="${1:-lines}"
FILTER='/mocks($|/)|/pbv1($|/)|/proto/v1($|/)'

PACKAGES="$(go list ./cmd/server/... ./internal/... | grep -Ev "$FILTER")"

case "$MODE" in
lines)
    printf '%s\n' "$PACKAGES"
    ;;
csv)
    printf '%s\n' "$PACKAGES" | paste -sd, -
    ;;
*)
    echo "unsupported mode: $MODE" >&2
    exit 1
    ;;
esac
