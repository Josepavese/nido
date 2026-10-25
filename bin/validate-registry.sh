#!/usr/bin/env bash
set -euo pipefail
# Isolate validation from all installed VM state and caches.
REGISTRY=${1:-registry/images.json}
FILTER=${2:-""}
validation_root=$(mktemp -d "${TMPDIR:-/tmp}/nido-registry.XXXXXX")
trap 'rm -rf "$validation_root"' EXIT
export NIDO_HOME="$validation_root/nido"
mkdir -p "$NIDO_HOME/images"
cp "$REGISTRY" "$NIDO_HOME/images/.catalog.json"
go run ./cmd/registry-validator --registry "$REGISTRY" --filter "$FILTER"
