#!/usr/bin/env bash
#
# Sweep both the core Nido repo and the GUI project build artifacts.
#
# Pass-through options:
#   -n, --dry-run        Show what would be removed without deleting anything.
#   -y, --yes            Skip confirmation prompts.
#   -g, --git-gc         Run git garbage collection (default: on).
#   --no-git-gc          Skip git garbage collection.
#   --shallow-current    Drop local history and keep only the current origin default branch (default: on).
#   --no-shallow-current Disable shallow reset.
#   --no-dist            Do not remove dist/ bundles.
#   --no-binaries        Do not remove built binaries (nido, bin/nido).
#   --no-registry-tools  Do not remove registry-builder / registry-validator.
#   -h, --help           Show this message.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
GUI_CLEAN="${REPO_ROOT}/gui/tools/housekeeping/clean.sh"

usage() {
  grep "^# " "${BASH_SOURCE[0]}" | sed 's/^# //'
  exit 0
}

for arg in "$@"; do
  case "$arg" in
    -h|--help) usage ;;
  esac
done

"${SCRIPT_DIR}/clean-nido.sh" "$@"

if [[ -x "$GUI_CLEAN" ]]; then
  "${GUI_CLEAN}" "$@"
else
  echo "GUI cleaner not found at ${GUI_CLEAN}; skipping GUI sweep."
fi
