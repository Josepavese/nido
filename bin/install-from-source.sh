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

# OS and Architecture Detection
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

if [[ $QEMU_MISSING -eq 0 ]]; then
    echo "  ${OK} QEMU binaries found."
else
    echo "  ${WARN} QEMU is missing. Nido needs it to hatch VMs."
    read -p "  📦 Would you like to install QEMU dependencies automatically? (y/N) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ "$OS" == "linux" ]]; then
            PKG="qemu-system-x86 qemu-utils"
            [[ "$ARCH" == "arm64" || "$ARCH" == "aarch64" ]] && PKG="qemu-system-arm qemu-utils"
            echo "  ${STEP} Updating repositories and installing ${PKG}..."
            sudo apt update && sudo apt install -y $PKG
        elif [[ "$OS" == "darwin" ]]; then
            if command -v brew >/dev/null 2>&1; then
                echo "  ${STEP} Installing QEMU via Homebrew..."
                brew install qemu
            else
                echo "  ${ERR} Homebrew not found. Please install Homebrew or QEMU manually."
            fi
        fi
    else
        QEMU_CMD="sudo apt update && sudo apt install qemu-system-x86 qemu-utils"
        [[ "$ARCH" == "arm64" || "$ARCH" == "aarch64" ]] && QEMU_CMD="sudo apt update && sudo apt install qemu-system-arm qemu-utils"
        echo "  ${INFO} Skipping automatic installation. You'll need to install it manually."
        echo "        Linux: ${CYAN}${QEMU_CMD}${RESET}"
        echo "        macOS: ${CYAN}brew install qemu${RESET}"
    fi
fi

# KVM Permissions (Linux Only)
if [[ "$OS" == "linux" && -e /dev/kvm && ! -w /dev/kvm ]]; then
    echo "  ${WARN} KVM detected but you don't have permission to use it."
    read -p "  🔐 Would you like to grant permissions to the current user? (y/N) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "  ${STEP} Adding $USER to 'kvm' group..."
        sudo usermod -aG kvm $USER
        echo "  ${OK} Permissions granted. You may need to logout or run 'newgrp kvm'."
    fi
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

# 5. Desktop Integration
echo ""
echo "${STEP} Finalizing Nesting (Desktop Integration)..."
cp "./resources/nido.png" "${NIDO_HOME}/nido.png"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [[ "$OS" == "linux" ]]; then
    DESKTOP_DIR="${HOME}/.local/share/applications"
    mkdir -p "$DESKTOP_DIR"
    # Create a small launcher script to handle dimensions
    cat > "${NIDO_HOME}/bin/nido-launcher" <<EOF
#!/bin/bash
# 🐣 Nido Linux Launcher with Optimal Dimensions
if command -v gnome-terminal >/dev/null 2>&1; then
    gnome-terminal --geometry=84x26 -- "${NIDO_HOME}/bin/nido" gui
elif command -v x-terminal-emulator >/dev/null 2>&1; then
    x-terminal-emulator -e "${NIDO_HOME}/bin/nido" gui
else
    "${NIDO_HOME}/bin/nido" gui
fi
EOF
    chmod +x "${NIDO_HOME}/bin/nido-launcher"

    cat > "${DESKTOP_DIR}/nido.desktop" <<EOF
[Desktop Entry]
Name=Nido
Comment=The Universal VM Nest
Exec=${NIDO_HOME}/bin/nido-launcher
Icon=${NIDO_HOME}/nido.png
Terminal=false
Type=Application
Categories=System;Utility;
Keywords=vm;qemu;virtual;nest;
EOF
    chmod +x "${DESKTOP_DIR}/nido.desktop"
    echo "  ${OK} Launcher entry created in ${DESKTOP_DIR}/nido.desktop"
elif [[ "$OS" == "darwin" ]]; then
    APP_DIR="${HOME}/Applications/Nido.app"
    mkdir -p "${APP_DIR}/Contents/MacOS"
    cat > "${APP_DIR}/Contents/MacOS/Nido" <<EOF
#!/bin/bash
# 🐣 Nido macOS Launcher
osascript -e 'tell application "Terminal"
    activate
    set newWin to (do script "${NIDO_HOME}/bin/nido gui")
    set bounds of window 1 of (application "Terminal") to {100, 100, 780, 580}
end tell'
EOF
    chmod +x "${APP_DIR}/Contents/MacOS/Nido"
    echo "  ${OK} Application bundle created in ${APP_DIR}"
fi

echo ""
echo "${BOLD}${GREEN}  CONGRATULATIONS! ${BIRD}${RESET}"
echo "  Nido v3 is now installed and ready to fly."
echo ""
echo "  ${BOLD}Next steps:${RESET}"
echo "  1. Reload your shell: ${CYAN}source ${SHELL_RC:-~/.bashrc}${RESET}"
echo "  2. Test Nido: ${CYAN}nido version${RESET}"
echo ""
