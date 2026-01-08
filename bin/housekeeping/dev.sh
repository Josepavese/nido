#!/usr/bin/env bash
#
# Nido dev helper: Build the current source and replace the locally installed binary.
#
# Usage:
#   bin/housekeeping/dev.sh
#

set -euo pipefail

# Path Resolution
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Styling
BOLD="$(tput bold || echo '')"
GREEN="$(tput setaf 2 || echo '')"
CYAN="$(tput setaf 6 || echo '')"
RED="$(tput setaf 1 || echo '')"
RESET="$(tput sgr0 || echo '')"

line_sep="${CYAN}${BOLD}┌────────────────────────────────────────────┐${RESET}"
line_mid="${CYAN}${BOLD}│${RESET}"
line_end="${CYAN}${BOLD}└────────────────────────────────────────────┘${RESET}"

header() {
    printf "\n%s\n" "$line_sep"
    printf "%s ${BOLD}🔨 Nido Dev Builder${RESET}                         %s\n" "$line_mid" "$line_mid"
    printf "%s\n" "$line_end"
}

info() {
    printf "${CYAN}➜${RESET} %s\n" "$1"
}

success() {
    printf "${GREEN}✔${RESET} %s\n" "$1"
}

error() {
    printf "${RED}✖${RESET} %s\n" "$1"
}

# --- Main ---

header

# 1. Locate current installation
info "Locating current nido installation..."
if ! TARGET=$(which nido); then
    error "nido not found in PATH."
    printf "  Please ensure you have a valid installation of nido first.\n"
    exit 1
fi
printf "  Found at: ${BOLD}%s${RESET}\n\n" "$TARGET"

# 2. Build
info "Building from source..."
if go build -C "${REPO_ROOT}" -o nido ./cmd/nido; then
    success "Build successful."
else
    error "Build failed."
    exit 1
fi

# 3. Replace
info "Replacing binary..."
mv "${REPO_ROOT}/nido" "$TARGET"
success "Binary replaced."

# 4. Verify
printf "\n"
nido version
printf "\n${GREEN}⚡ Ready to fly.${RESET}\n"
