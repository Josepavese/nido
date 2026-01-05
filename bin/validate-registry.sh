#!/bin/bash
set -e

# Wrapper to run the registry validator
# Usage: bin/validate-registry.sh [path/to/registry.json]

REGISTRY=${1:-registry/images.json}
FILTER=${2:-""}

echo "Starting Nido Registry Validation..."
echo "Using registry: $REGISTRY"
if [ -n "$FILTER" ]; then
    echo "Filtering by: $FILTER"
fi

echo "Syncing local registry to ~/.nido/images/.catalog.json..."
mkdir -p ~/.nido/images
cp "$REGISTRY" ~/.nido/images/.catalog.json

go run cmd/registry-validator/main.go --registry "$REGISTRY" --filter "$FILTER"
