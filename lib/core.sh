#!/usr/bin/env bash
#
# nido - Shared Core Utilities
#

# --- Colors & Icons (Exported) ---
export BOLD="$(tput bold 2>/dev/null || echo '')"
export BLUE="$(tput setaf 4 2>/dev/null || echo '')"
export GREEN="$(tput setaf 2 2>/dev/null || echo '')"
export YELLOW="$(tput setaf 3 2>/dev/null || echo '')"
export RED="$(tput setaf 1 2>/dev/null || echo '')"
export CYAN="$(tput setaf 6 2>/dev/null || echo '')"
export PURPLE="$(tput setaf 5 2>/dev/null || echo '')"
export RESET="$(tput sgr0 2>/dev/null || echo '')"

export INFO="[${BLUE}i${RESET}]"
export OK="[${GREEN}ok${RESET}]"
export WARN="[${YELLOW}!${RESET}]"
export ERR="[${RED}ERR${RESET}]"
export BIRD="🐣"

# --- Utils ---

# Checks if required commands exist
require_cmds() {
  for cmd in "$@"; do
    command -v "$cmd" >/dev/null 2>&1 || {
      echo "Missing required command: $cmd" >&2
      exit 1
    }
  done
}

# Loads configuration from env or default path
load_config() {
  local default_cfg="$1"
  local cfg="${VMOPS_CONFIG:-$default_cfg}"
  if [[ -f "$cfg" ]]; then
    set -a
    # shellcheck disable=SC1090
    . "$cfg"
    set +a
  fi
}

# Sets strict defaults for all configuration variables
load_defaults() {
  export TEMPLATE_DEFAULT="${TEMPLATE_DEFAULT:-template-headless}"
  export VMS_POOL="${VMS_POOL:-vms}"
  export WAIT_TIMEOUT="${WAIT_TIMEOUT:-60}"
  export POOL_PATH="${POOL_PATH:-/var/lib/libvirt/images}"
  export BACKUP_DIR="${BACKUP_DIR:-$POOL_PATH/backups}"
  export VM_MEM_MB="${VM_MEM_MB:-2048}"
  export VM_VCPUS="${VM_VCPUS:-2}"
  export VM_OS_VARIANT="${VM_OS_VARIANT:-debian12}"
  export NETWORK_HOSTONLY="${NETWORK_HOSTONLY:-hostonly56}"
  export NETWORK_NAT="${NETWORK_NAT:-default}"
  export SSH_USER="${SSH_USER:-vmuser}"
  export VM_NESTED="${VM_NESTED:-false}"
  export GRAPHICS="${GRAPHICS:-spice}"
}

# Spinner utility (Blocking, prints to stderr to avoid polluting stdout in MCP)
spinner() {
  local pid=$1
  local msg=$2
  local delay=0.1
  local spinstr='|/-\'
  
  # Only show spinner if stderr is a TTY
  if [ -t 2 ]; then
      printf "  ${BOLD}${CYAN}..${RESET} %s" "$msg" >&2
      while kill -0 "$pid" 2>/dev/null; do
        local temp=${spinstr#?}
        printf " [%c]" "$spinstr" >&2
        local spinstr=$temp${spinstr%"$temp"}
        sleep $delay
        printf "\b\b\b\b" >&2
      done
      printf "\r${OK} %s    \n" "$msg" >&2
  else
      # If not a TTY (e.g. MCP stdio), just wait
      wait "$pid"
  fi
}

# Helper to extract IP from text block
extract_ip() {
  grep -oE '([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -vE '^127\.' | head -n 1 || true
}
