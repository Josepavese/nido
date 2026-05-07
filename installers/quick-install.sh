#!/usr/bin/env bash
# 🪺 Nido Quick Installer - Ultra Lightweight Edition
# Downloads only the binary. No repo cloning. Lightning fast.
set -euo pipefail

# --- Colors & Icons ---
BOLD="$(tput bold 2>/dev/null || echo '')"
GREEN="$(tput setaf 2 2>/dev/null || echo '')"
CYAN="$(tput setaf 6 2>/dev/null || echo '')"
YELLOW="$(tput setaf 3 2>/dev/null || echo '')"
RED="$(tput setaf 1 2>/dev/null || echo '')"
MAGENTA="$(tput setaf 5 2>/dev/null || echo '')"
RESET="$(tput sgr0 2>/dev/null || echo '')"

echo "${BOLD}${CYAN}"
echo "  🪺 Nido Quick Install"
echo "  Lightning-fast VM management"
echo "${RESET}"

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Detect Termux
IS_TERMUX=0
if [ -d "/data/data/com.termux" ]; then
  IS_TERMUX=1
fi

case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  *) echo "${RED}❌ Unsupported OS: $OS${RESET}"; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "${RED}❌ Unsupported architecture: $ARCH${RESET}"; exit 1 ;;
esac

case "${OS}/${ARCH}" in
  linux/amd64|linux/arm64|darwin/amd64|darwin/arm64) ;;
  *)
    echo "${RED}❌ No pre-built release artifact for ${OS}/${ARCH}.${RESET}"
    echo "   Use the source installer instead:"
    echo "   curl -fsSL https://raw.githubusercontent.com/Josepavese/nido/main/installers/build-from-source.sh | bash"
    exit 1
    ;;
esac

# Determine latest release
echo "${CYAN}🔍 Fetching latest release...${RESET}"
LATEST_RELEASE="${NIDO_VERSION:-}"
if [ -z "$LATEST_RELEASE" ]; then
  LATEST_RELEASE=$(curl -sL https://api.github.com/repos/Josepavese/nido/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

if [ -z "$LATEST_RELEASE" ]; then
  echo "${RED}❌ Failed to fetch latest release${RESET}"
  exit 1
fi

echo "${GREEN}✅ Latest version: ${LATEST_RELEASE}${RESET}"

ASSET_NAME="nido-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/Josepavese/nido/releases/download/${LATEST_RELEASE}/${ASSET_NAME}"
CHECKSUM_URL="https://github.com/Josepavese/nido/releases/download/${LATEST_RELEASE}/SHA256SUMS"

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/nido-install.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT
ARCHIVE_PATH="${TMP_DIR}/${ASSET_NAME}"
CHECKSUM_PATH="${TMP_DIR}/SHA256SUMS"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    return 1
  fi
}

verify_release_checksum() {
  if ! curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM_PATH"; then
    echo "${YELLOW}⚠️  SHA256SUMS not available for ${LATEST_RELEASE}; skipping archive checksum verification.${RESET}"
    return 0
  fi
  expected="$(awk -v asset="$ASSET_NAME" '{name=$2; sub(/^\*/, "", name); if (name == asset) print $1}' "$CHECKSUM_PATH")"
  if [ -z "$expected" ]; then
    echo "${YELLOW}⚠️  ${ASSET_NAME} not listed in SHA256SUMS; skipping archive checksum verification.${RESET}"
    return 0
  fi
  actual="$(sha256_file "$ARCHIVE_PATH")" || {
    echo "${YELLOW}⚠️  No local SHA-256 tool found; skipping archive checksum verification.${RESET}"
    return 0
  }
  if [ "$actual" != "$expected" ]; then
    echo "${RED}❌ Archive checksum mismatch for ${ASSET_NAME}${RESET}"
    exit 1
  fi
  echo "${GREEN}✅ Archive checksum verified${RESET}"
}

echo "${CYAN}📥 Downloading ${ASSET_NAME}...${RESET}"

if ! curl -fsSL "$DOWNLOAD_URL" -o "$ARCHIVE_PATH"; then
  echo "${RED}❌ Download failed${RESET}"
  exit 1
fi
verify_release_checksum

tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"
PKG_DIR="${TMP_DIR}/nido-${OS}-${ARCH}"
if [ ! -x "${PKG_DIR}/nido" ]; then
  echo "${RED}❌ Release archive does not contain nido${RESET}"
  exit 1
fi

# Setup Nido home directory
NIDO_HOME="${HOME}/.nido"
mkdir -p "${NIDO_HOME}/bin"
mkdir -p "${NIDO_HOME}/vms"
mkdir -p "${NIDO_HOME}/run"
mkdir -p "${NIDO_HOME}/images"
mkdir -p "${NIDO_HOME}/registry"

install -m 0755 "${PKG_DIR}/nido" "${NIDO_HOME}/bin/nido"
if [ -x "${PKG_DIR}/nido-validator" ]; then
  install -m 0755 "${PKG_DIR}/nido-validator" "${NIDO_HOME}/bin/nido-validator"
fi
if [ -d "${PKG_DIR}/registry" ]; then
  cp -R "${PKG_DIR}/registry/." "${NIDO_HOME}/registry/"
fi

echo "${GREEN}✅ Binary installed to ${NIDO_HOME}/bin/nido${RESET}"

# Download default themes
THEMES_URL="https://raw.githubusercontent.com/Josepavese/nido/main/resources/themes.json"
echo "${CYAN}🎨 Fetching visual themes...${RESET}"
if curl -fsSL "$THEMES_URL" -o "${NIDO_HOME}/themes.json"; then
  echo "${GREEN}✅ Themes installed to ${NIDO_HOME}/themes.json${RESET}"
else
  echo "${YELLOW}⚠️ Failed to download themes (skipped)${RESET}"
fi

# Create default config if missing
if [ ! -f "${NIDO_HOME}/config.env" ]; then
  cat > "${NIDO_HOME}/config.env" << 'EOF'
# Nido Configuration
BACKUP_DIR=${HOME}/.nido/backups
TEMPLATE_DEFAULT=template-headless
SSH_USER=vmuser
EOF
  mkdir -p "${NIDO_HOME}/backups"
  echo "${GREEN}✅ Default config created${RESET}"
fi

# Update PATH
SHELL_RC=""
case "$SHELL" in
  */bash) SHELL_RC="$HOME/.bashrc" ;;
  */zsh)  SHELL_RC="$HOME/.zshrc" ;;
esac

if [ -n "$SHELL_RC" ] && [ -f "$SHELL_RC" ]; then
  # Update PATH
  if ! grep -q "${NIDO_HOME}/bin" "$SHELL_RC" 2>/dev/null; then
    echo "" >> "$SHELL_RC"
    echo "# Nido VM Manager" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:${NIDO_HOME}/bin\"" >> "$SHELL_RC"
    echo "${GREEN}✅ Added to PATH in ${SHELL_RC}${RESET}"
  fi

  # Setup Completions
  case "$SHELL" in
    */bash)
      "${NIDO_HOME}/bin/nido" completion bash > "${NIDO_HOME}/bin/nido.bash"
      if ! grep -q "nido.bash" "$SHELL_RC" 2>/dev/null; then
        echo "source \"${NIDO_HOME}/bin/nido.bash\"" >> "$SHELL_RC"
        echo "${GREEN}✅ Bash completions enabled${RESET}"
      fi
      ;;
    */zsh)
      "${NIDO_HOME}/bin/nido" completion zsh > "${NIDO_HOME}/bin/nido.zsh"
      if ! grep -q "nido.zsh" "$SHELL_RC" 2>/dev/null; then
        echo "source \"${NIDO_HOME}/bin/nido.zsh\"" >> "$SHELL_RC"
        echo "${GREEN}✅ Zsh completions enabled${RESET}"
      fi
      ;;
  esac
fi

# Desktop Integration
echo "${CYAN}🎨 Setting up Desktop Integration...${RESET}"
# Download icon if possible, or use a default. For quick install, we can skip icon or use a generic one if we don't want to bundle it.
# However, we can try to download it from the repo.
ICON_URL="https://raw.githubusercontent.com/Josepavese/nido/main/resources/nido.png"
if curl -fsSL "$ICON_URL" -o "${NIDO_HOME}/nido.png"; then
    echo "${GREEN}✅ Icon downloaded${RESET}"
else
    # Fallback to chick emoji if download fails (not really possible as icon, but better than nothing)
    echo "${YELLOW}⚠️ Could not download icon, using generic${RESET}"
fi

if [ "$OS" = "linux" ] && [ $IS_TERMUX -eq 0 ]; then
    DESKTOP_DIR="${HOME}/.local/share/applications"
    mkdir -p "$DESKTOP_DIR"
    
    # Create compact launcher wrapper
    cat > "${NIDO_HOME}/bin/nido-launcher" <<EOF
#!/bin/bash
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
EOF
    chmod +x "${DESKTOP_DIR}/nido.desktop"
    echo "${GREEN}✅ Launcher entry created${RESET}"
elif [ "$OS" = "darwin" ]; then
    APP_DIR="${HOME}/Applications/Nido.app"
    mkdir -p "${APP_DIR}/Contents/MacOS"
    cat > "${APP_DIR}/Contents/MacOS/Nido" <<EOF
#!/bin/bash
osascript -e 'tell application "Terminal"
    activate
    set newWin to (do script "${NIDO_HOME}/bin/nido gui")
    set bounds of window 1 of (application "Terminal") to {100, 100, 780, 580}
end tell'
EOF
    chmod +x "${APP_DIR}/Contents/MacOS/Nido"
    echo "${GREEN}✅ Application bundle created${RESET}"
fi

# --- Dependency Check & Proactive Install ---
echo "${CYAN}🔍 Checking flight readiness (dependencies)...${RESET}"
QEMU_INSTALLED=0
if command -v qemu-system-x86_64 >/dev/null 2>&1 || command -v qemu-system-aarch64 >/dev/null 2>&1 || command -v qemu-system >/dev/null 2>&1; then
    QEMU_INSTALLED=1
fi

ISO_TOOL_INSTALLED=0
if command -v cloud-localds >/dev/null 2>&1 || command -v genisoimage >/dev/null 2>&1 || command -v mkisofs >/dev/null 2>&1 || command -v xorriso >/dev/null 2>&1; then
    ISO_TOOL_INSTALLED=1
fi

if [ $QEMU_INSTALLED -eq 0 ] || [ $ISO_TOOL_INSTALLED -eq 0 ]; then
    echo "${YELLOW}⚠️  Missing dependencies. Nido needs QEMU and ISO tools (cloud-utils/genisoimage).${RESET}"
    if [ $QEMU_INSTALLED -eq 0 ]; then echo "   - QEMU: Missing"; else echo "   - QEMU: OK"; fi
    if [ $ISO_TOOL_INSTALLED -eq 0 ]; then echo "   - ISO Tools: Missing"; else echo "   - ISO Tools: OK"; fi
    
    read -p "📦 Would you like to install dependencies automatically? (y/N) " -n 1 -r < /dev/tty
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [ $IS_TERMUX -eq 1 ]; then
            PKG="qemu-system-x86-64-headless qemu-utils xorriso"
            echo "${CYAN}🛠️  Installing ${PKG}...${RESET}"
            pkg install -y $PKG
        elif [ "$OS" = "linux" ]; then
            PKG="qemu-system-x86 qemu-utils cloud-utils genisoimage"
            [ "$ARCH" = "arm64" ] && PKG="qemu-system-arm qemu-system-gui qemu-utils cloud-utils genisoimage"
            echo "${CYAN}🛠️  Updating repositories and installing ${PKG}...${RESET}"
            sudo apt update && sudo apt install -y $PKG
        elif [ "$OS" = "darwin" ]; then
            if command -v brew >/dev/null 2>&1; then
                echo "${CYAN}🛠️  Installing QEMU & cdrtools via Homebrew...${RESET}"
                brew install qemu cdrtools
            else
                echo "${RED}❌ Homebrew not found. Please install Homebrew manually.${RESET}"
            fi
        fi
    else
        echo "${YELLOW}⚠️  Skipping automatic installation. You'll need to install them manually.${RESET}"
    fi
else
    echo "${GREEN}✅ Dependencies (QEMU & ISO tools) are ready.${RESET}"
fi

# KVM Permissions (Linux Only)
if [ "$OS" = "linux" ] && [ $IS_TERMUX -eq 0 ]; then
    echo "${CYAN}🔍 Checking KVM accessibility...${RESET}"
    if [ -e /dev/kvm ]; then
        # Check if already in group or have permissions
        if [ ! -w /dev/kvm ] && ! groups | grep -q "\bkvm\b"; then
            echo "${YELLOW}⚠️  KVM detected but you don't have permission to use it.${RESET}"
            echo -n "🔐 Would you like to grant permissions to the current user? (y/N) " > /dev/tty
            read -n 1 -r RESPONSE < /dev/tty
            echo ""
            if [[ "$RESPONSE" =~ ^[Yy]$ ]]; then
                echo "${CYAN}🛠️  Adding $USER to 'kvm' group...${RESET}"
                sudo usermod -aG kvm "$USER"
                echo ""
                echo "${BOLD}${MAGENTA}🚨 IMPORTANT: SESSION RESTART REQUIRED${RESET}"
                echo "To apply nested virtualization permissions, you MUST either:"
                echo "  • Restart your terminal session (logout and login)"
                echo "  • Run ${CYAN}newgrp kvm${RESET} in the current terminal before launching Nido."
                echo ""
                echo "${GREEN}✅ Permissions granted.${RESET}"
            fi
        else
            echo "${GREEN}✅ KVM is accessible.${RESET}"
        fi
    else
        echo "${YELLOW}ℹ️  KVM not found. Nested virtualization might be disabled in host.${RESET}"
    fi
fi

# --- Final Tip ---
echo ""
echo "${BOLD}${GREEN}🎉 Installation complete!${RESET}"
echo ""
echo "${BOLD}Next steps:${RESET}"
echo "  1. Reload shell:  ${CYAN}source ${SHELL_RC:-~/.bashrc}${RESET}"
echo "  2. Verify install: ${CYAN}nido version${RESET}"
echo "  3. Check system:   ${CYAN}nido doctor${RESET}"
echo ""
if command -v qemu-system-x86_64 >/dev/null 2>&1 || command -v qemu-system-aarch64 >/dev/null 2>&1 || command -v qemu-system >/dev/null 2>&1; then
    if [ "$OS" = "linux" ] && [ $IS_TERMUX -eq 0 ] && [ ! -w /dev/kvm ]; then
        echo "${YELLOW}⚠️  KVM needs permission: sudo usermod -aG kvm \$USER && newgrp kvm${RESET}"
    else
        echo "${GREEN}✨ QEMU is ready for liftoff!${RESET}"
    fi
else
    QEMU_CMD="sudo apt update && sudo apt install qemu-system-x86 qemu-utils cloud-utils genisoimage"
    [ "$ARCH" = "arm64" ] && QEMU_CMD="sudo apt update && sudo apt install qemu-system-arm qemu-system-gui qemu-utils cloud-utils genisoimage"
    echo "${YELLOW}💡 Note: You still need QEMU & ISO tools to run VMs${RESET}"
    echo "   Linux: ${CYAN}${QEMU_CMD}${RESET}"
    [ $IS_TERMUX -eq 1 ] && echo "   Termux: ${CYAN}pkg install qemu-system-x86-64-headless qemu-utils xorriso${RESET}"
    echo "   macOS: ${CYAN}brew install qemu cdrtools${RESET}"
fi
echo ""
echo "${BOLD}\"It's not a VM, it's a lifestyle.\" 🪺${RESET}"
