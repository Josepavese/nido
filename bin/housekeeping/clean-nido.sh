#!/usr/bin/env bash
#
# Nido repository housekeeping: remove local build/release artifacts and compact git history.
#
# Options:
#   -n, --dry-run            Show what would be removed without deleting anything.
#   -y, --yes                Skip confirmation prompts (disables interactive UI).
#   --ui / --no-ui           Force enable/disable the interactive UI (default: on when TTY).
#   -g, --git-gc             Run git garbage collection (default: on).
#   --no-git-gc              Skip git garbage collection.
#   --shallow-current        Drop local history and keep only the current origin default branch (default: on).
#   --no-shallow-current     Disable shallow reset.
#   --allow-tracked          Permit removal of tracked files in the targets list (default: skip tracked files).
#   --no-dist                Do not remove dist/ bundles.
#   --no-binaries            Do not remove built binaries (nido, bin/nido).
#   --no-registry-tools      Do not remove registry-builder / registry-validator.
#   -h, --help               Show this message.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
DRY_RUN=0
AUTO_YES=0
INTERACTIVE=1
RUN_GC=1
RUN_SHALLOW=1
SKIP_DIST=0
SKIP_BIN=0
SKIP_REGISTRY=0
ALLOW_TRACKED=0
STASH_REF=""

usage() {
  grep "^# " "${BASH_SOURCE[0]}" | sed 's/^# //'
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    -n|--dry-run) DRY_RUN=1 ;;
    -y|--yes) AUTO_YES=1 ;;
    --ui) INTERACTIVE=1 ;;
    --no-ui) INTERACTIVE=0 ;;
    -g|--git-gc) RUN_GC=1 ;;
    --no-git-gc) RUN_GC=0 ;;
    --shallow-current) RUN_SHALLOW=1 ;;
    --no-shallow-current) RUN_SHALLOW=0 ;;
    --allow-tracked) ALLOW_TRACKED=1 ;;
    --no-dist) SKIP_DIST=1 ;;
    --no-binaries) SKIP_BIN=1 ;;
    --no-registry-tools) SKIP_REGISTRY=1 ;;
    -h|--help) usage ;;
    *) echo "Unknown option: $1" >&2; usage ;;
  esac
  shift
done

if [[ $AUTO_YES -eq 1 ]]; then
  INTERACTIVE=0
fi
if [[ ! -t 0 ]]; then
  INTERACTIVE=0
fi

render_menu() {
  local clean_dist=$1 clean_bin=$2 clean_registry=$3 shallow=$4 gc=$5 allow=$6
  local BOLD BLUE CYAN MAGENTA GREEN RED RESET
  BOLD="$(tput bold || echo '')"
  BLUE="$(tput setaf 4 || echo '')"
  CYAN="$(tput setaf 6 || echo '')"
  MAGENTA="$(tput setaf 5 || echo '')"
  GREEN="$(tput setaf 2 || echo '')"
  RED="$(tput setaf 1 || echo '')"
  RESET="$(tput sgr0 || echo '')"

  local line_sep="${BLUE}${BOLD}┌────────────────────────────────────────────┐${RESET}"
  local line_mid="${BLUE}${BOLD}│${RESET}"
  local line_end="${BLUE}${BOLD}└────────────────────────────────────────────┘${RESET}"

  status_icon() {
    local enabled="$1"
    if [[ "$enabled" -eq 1 ]]; then
      printf "${GREEN}●${RESET}"
    else
      printf "${RED}○${RESET}"
    fi
  }

  row() {
    local enabled="$1" label="$2" key="$3" desc="$4"
    printf "%s\t${BOLD}%-14s${RESET}\t[%s] %s\n" "$(status_icon "$enabled")" "$label" "$key" "$desc"
  }

  printf "\033c"
  printf "%s\n" "$line_sep"
  printf "%s ${BOLD}🪺  Nido Housekeeper · payload selector${RESET} %s\n" "$line_mid" "$line_mid"
  printf "%s\n" "$line_end"
  printf "${CYAN}Use 1-6 to toggle · a = toggle all · Enter = launch · q = abort${RESET}\n\n"

  row "$clean_dist"     "dist bundles"    "1" "Remove ./dist"
  row "$clean_bin"      "binaries"        "2" "Remove nido, bin/nido"
  row "$clean_registry" "registry tools"  "3" "Remove registry-builder / registry-validator"
  row "$shallow"        "shallow reset"   "4" "Depth=1 reset to origin default (stash/apply changes)"
  row "$gc"             "git gc"          "5" "git gc --prune=now --aggressive"
  row "$allow"          "keep tracked?"   "6" "Permit deleting tracked targets (⚠)"

  printf "\n${MAGENTA}Tip:${RESET} Defaults mirror the CLI flags; flip what you don't need.\n"
}

if [[ $INTERACTIVE -eq 1 ]]; then
  clean_dist=$((SKIP_DIST==0))
  clean_bin=$((SKIP_BIN==0))
  clean_registry=$((SKIP_REGISTRY==0))
  shallow=$((RUN_SHALLOW==1))
  gc=$((RUN_GC==1))
  allow=$((ALLOW_TRACKED==1))

  while true; do
    render_menu "$clean_dist" "$clean_bin" "$clean_registry" "$shallow" "$gc" "$allow"
    read -rsn1 key
    case "$key" in
      1) clean_dist=$((1-clean_dist)) ;;
      2) clean_bin=$((1-clean_bin)) ;;
      3) clean_registry=$((1-clean_registry)) ;;
      4) shallow=$((1-shallow)) ;;
      5) gc=$((1-gc)) ;;
      6) allow=$((1-allow)) ;;
      a|A) # toggle all
        clean_dist=$((1-clean_dist))
        clean_bin=$((1-clean_bin))
        clean_registry=$((1-clean_registry))
        shallow=$((1-shallow))
        gc=$((1-gc))
        allow=$((1-allow))
        ;;
      q|Q) echo "Aborted."; exit 1 ;;
      "") break ;; # Enter sends empty string with -n1 -s
      $'\n') break ;;
      *) ;;
    esac
  done

  SKIP_DIST=$((clean_dist ? 0 : 1))
  SKIP_BIN=$((clean_bin ? 0 : 1))
  SKIP_REGISTRY=$((clean_registry ? 0 : 1))
  RUN_SHALLOW=$((shallow ? 1 : 0))
  RUN_GC=$((gc ? 1 : 0))
  ALLOW_TRACKED=$((allow ? 1 : 0))
fi

existing_targets=()
maybe_add() {
  local path="$1"
  local skip="$2"
  if [[ "$skip" -eq 0 && -e "$path" ]]; then
    existing_targets+=("$path")
  fi
}

maybe_add "${REPO_ROOT}/dist" "$SKIP_DIST"
maybe_add "${REPO_ROOT}/nido" "$SKIP_BIN"
maybe_add "${REPO_ROOT}/bin/nido" "$SKIP_BIN"
maybe_add "${REPO_ROOT}/registry-builder" "$SKIP_REGISTRY"
maybe_add "${REPO_ROOT}/registry-validator" "$SKIP_REGISTRY"

if [[ ${#existing_targets[@]} -eq 0 && $RUN_GC -eq 0 && $RUN_SHALLOW -eq 0 ]]; then
  echo "Nothing to clean: no build artifacts found."
  exit 0
fi

if [[ $AUTO_YES -eq 0 ]]; then
  if [[ ${#existing_targets[@]} -gt 0 ]]; then
    echo "The following Nido build artifacts will be removed:"
    for path in "${existing_targets[@]}"; do
      echo " - ${path}"
    done
  fi
  if [[ $RUN_GC -eq 1 ]]; then
    echo " - git gc --prune=now --aggressive (repack .git to reclaim space)"
  fi
  if [[ $RUN_SHALLOW -eq 1 ]]; then
    echo " - Shallow reset to origin default branch (drops local git history)"
  fi
  read -r -p "Proceed? [y/N] " reply
  if [[ ! "$reply" =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
  fi
fi

for path in "${existing_targets[@]}"; do
  if [[ $ALLOW_TRACKED -eq 0 ]] && git -C "${REPO_ROOT}" ls-files --error-unmatch "$path" >/dev/null 2>&1; then
    echo "Skipping tracked file ${path} (use --allow-tracked to remove)."
    continue
  fi
  if [[ $DRY_RUN -eq 1 ]]; then
    echo "[dry-run] Would remove ${path}"
  else
    rm -rf -- "$path"
    echo "Removed ${path}"
  fi
done

if [[ $RUN_SHALLOW -eq 1 ]]; then
  if [[ $DRY_RUN -eq 1 ]]; then
    if [[ -n "$(git -C "${REPO_ROOT}" status --porcelain)" ]]; then
      echo "[dry-run] Working tree dirty: would stash changes (including untracked), shallow-reset, then reapply stash (may conflict)."
    fi
    echo "[dry-run] Would shallow-reset to origin default branch (fetch --depth=1, reset --hard, prune reflog, aggressive gc)."
  else
    if [[ -n "$(git -C "${REPO_ROOT}" status --porcelain)" ]]; then
      echo "Working tree dirty: stashing changes (including untracked) before shallow reset..."
      git -C "${REPO_ROOT}" stash push -u -m "housekeeping-shallow-$(date +%Y%m%d%H%M%S)" >/dev/null
      STASH_REF="$(git -C "${REPO_ROOT}" stash list -n1 | head -n1 | cut -d: -f1 | tr -d '[:space:]')"
      echo "Stashed as ${STASH_REF:-stash@{0}}"
    fi
    DEFAULT_REF="$(git -C "${REPO_ROOT}" symbolic-ref --quiet refs/remotes/origin/HEAD || true)"
    DEFAULT_BRANCH="${DEFAULT_REF##refs/remotes/origin/}"
    if [[ -z "$DEFAULT_BRANCH" ]]; then
      DEFAULT_BRANCH="main"
    fi
    echo "Shallow fetching origin/${DEFAULT_BRANCH} (depth=1)..."
    git -C "${REPO_ROOT}" fetch --depth=1 origin "${DEFAULT_BRANCH}"
    echo "Resetting working tree to origin/${DEFAULT_BRANCH}..."
    git -C "${REPO_ROOT}" checkout -B "${DEFAULT_BRANCH}" "origin/${DEFAULT_BRANCH}"
    git -C "${REPO_ROOT}" reset --hard "origin/${DEFAULT_BRANCH}"
    echo "Expiring reflog and pruning old objects..."
    git -C "${REPO_ROOT}" reflog expire --expire=now --all
    git -C "${REPO_ROOT}" gc --prune=now --aggressive
    if [[ -n "$STASH_REF" ]]; then
      echo "Reapplying stashed changes (${STASH_REF})..."
      if git -C "${REPO_ROOT}" stash apply "$STASH_REF"; then
        git -C "${REPO_ROOT}" stash drop "$STASH_REF" >/dev/null || true
      else
        echo "⚠️  Stash apply failed. Stash remains as ${STASH_REF}. Resolve conflicts and apply manually:"
        echo "    git -C \"${REPO_ROOT}\" stash apply ${STASH_REF}"
        exit 1
      fi
    fi
  fi
fi

if [[ $RUN_GC -eq 1 ]]; then
  if [[ $DRY_RUN -eq 1 ]]; then
    echo "[dry-run] Would run: git -C \"${REPO_ROOT}\" gc --prune=now --aggressive"
  else
    echo "Running git gc to compact repository objects..."
    git -C "${REPO_ROOT}" gc --prune=now --aggressive
  fi
fi

if [[ $DRY_RUN -eq 1 ]]; then
  echo "Dry run complete."
else
  echo "Nido workspace cleaned."
fi
