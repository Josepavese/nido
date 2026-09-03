#!/usr/bin/env bash

set -euo pipefail

readonly max_attempts=3

attempt=1
while [ "$attempt" -le "$max_attempts" ]; do
    if go mod download; then
        exit 0
    fi

    if [ "$attempt" -eq "$max_attempts" ]; then
        echo "Go module download failed after ${max_attempts} attempts." >&2
        exit 1
    fi

    delay=$((attempt * 10))
    echo "Go module download attempt ${attempt} failed; retrying in ${delay}s..." >&2
    sleep "$delay"
    attempt=$((attempt + 1))
done
