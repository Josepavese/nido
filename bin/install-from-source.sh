#!/usr/bin/env bash
#
# 🪺 Nido Developer Installer (Build from Source)
# For contributors and developers who want to build Nido from source code.
# End users should use: installers/quick-install.sh
#
set -euo pipefail

# --- Colors & Icons ---
BOLD="$(tput bold || echo '')"
BLUE="$(tput setaf 4 || echo '')"
GREEN="$(tput setaf 2 || echo '')"
YELLOW="$(tput setaf 3 || echo '')"
RED="$(tput setaf 1 || echo '')"
CYAN="$(tput setaf 6 || echo '')"
RESET="$(tput sgr0 || echo '')"

INFO="[${BLUE}i${RESET}]"
STEP="[${CYAN}step${RESET}]"
OK="[${GREEN}ok${RESET}]"
WARN="[${YELLOW}!${RESET}]"
ERR="[${RED}ERR${RESET}]"
BIRD="🐣"

echo "${BOLD}${BLUE}  Nido v3 (Go) Installation${RESET}"
echo "  ------------------------------------"

# 1. Check for Go
echo "${STEP} Checking for Go compiler..."
if ! command -v go >/dev/null 2>&1; then
    echo "  ${ERR} Go is missing! Please install Go (1.21+) to build Nido v3."
    exit 1
fi
echo "  ${OK} Go found: $(go version)"

# 1b. Check for QEMU (Runtime Dependency)
echo "${STEP} Checking for QEMU binaries (runtime)..."
QEMU_MISSING=0
if ! command -v qemu-system-x86_64 >/dev/null 2>&1; then
    echo "  ${WARN} qemu-system-x86_64 is missing! (Required to run VMs)"
    QEMU_MISSING=1
fi
if ! command -v qemu-img >/dev/null 2>&1; then
    echo "  ${WARN} qemu-img is missing! (Required for disk operations)"
    QEMU_MISSING=1
fi

if [[ $QEMU_MISSING -eq 0 ]]; then
    echo "  ${OK} QEMU binaries found."
else
    echo "  ${INFO} Note: You'll need to install QEMU before spawning VMs."
    echo "        Linux: sudo apt install qemu-system-x86 qemu-utils"
    echo "        macOS: brew install qemu"
fi

# 2. Build
echo ""
echo "${STEP} Building the new engine..."
go build -o nido ./cmd/nido
echo "  ${OK} Binary built successfully."

# 3. Setup local environment
echo ""
echo "${STEP} Structuring the nest (~/.nido)..."
NIDO_HOME="${HOME}/.nido"
mkdir -p "${NIDO_HOME}/bin"
mkdir -p "${NIDO_HOME}/run"
mkdir -p "${NIDO_HOME}/vms"
mkdir -p "${NIDO_HOME}/config"

mv nido "${NIDO_HOME}/bin/nido"

# Ensure config exists
if [[ ! -f "${NIDO_HOME}/config.env" ]]; then
    if [[ -f "./config/config.env" ]]; then
        cp "./config/config.env" "${NIDO_HOME}/config.env"
        echo "  ${OK} Default configuration copied to ${NIDO_HOME}/config.env"
    fi
fi

# 4. PATH check
echo ""
echo "${STEP} Checking your flight path (PATH)..."
SHELL_RC=""
case "$SHELL" in
  */bash) SHELL_RC="$HOME/.bashrc" ;;
  */zsh)  SHELL_RC="$HOME/.zshrc" ;;
esac

BIN_PATH="${NIDO_HOME}/bin"
if [[ -n "$SHELL_RC" ]] && [[ -f "$SHELL_RC" ]]; then
    if ! grep -q "$BIN_PATH" "$SHELL_RC"; then
        echo "  ${INFO} Adding Nido to ${SHELL_RC}..."
        echo "" >> "$SHELL_RC"
        echo "# Nido v3" >> "$SHELL_RC"
        echo "export PATH=\"\$PATH:$BIN_PATH\"" >> "$SHELL_RC"
        echo "  ${OK} PATH updated."
    else
        echo "  ${OK} Nido is already in your PATH."
    fi

    echo "${STEP} Setting up shell completion..."
    if [[ "$SHELL_RC" == *".bashrc" ]]; then
        if ! grep -q "nido completion bash" "$SHELL_RC"; then
            echo 'source <(nido completion bash)' >> "$SHELL_RC"
            echo "  ${OK} Bash completion added."
        fi
    elif [[ "$SHELL_RC" == *".zshrc" ]]; then
        if ! grep -q "nido completion zsh" "$SHELL_RC"; then
            echo 'source <(nido completion zsh)' >> "$SHELL_RC"
            echo "  ${OK} Zsh completion added."
        fi
    fi
fi

echo ""
echo "${BOLD}${GREEN}  CONGRATULATIONS! ${BIRD}${RESET}"
echo "  Nido v3 is now installed and ready to fly."
echo ""
echo "  ${BOLD}Next steps:${RESET}"
echo "  1. Reload your shell: ${CYAN}source ${SHELL_RC:-~/.bashrc}${RESET}"
echo "  2. Test Nido: ${CYAN}nido version${RESET}"
echo ""
