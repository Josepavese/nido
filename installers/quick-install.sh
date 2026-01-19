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
RESET="$(tput sgr0 2>/dev/null || echo '')"

echo "${BOLD}${CYAN}"
echo "  🪺 Nido Quick Install"
echo "  Lightning-fast VM management"
echo "${RESET}"

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

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

# Determine latest release
echo "${CYAN}🔍 Fetching latest release...${RESET}"
LATEST_RELEASE=$(curl -sL https://api.github.com/repos/Josepavese/nido/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
  echo "${RED}❌ Failed to fetch latest release${RESET}"
  exit 1
fi

echo "${GREEN}✅ Latest version: ${LATEST_RELEASE}${RESET}"

# Build download URL
BINARY_NAME="nido-${OS}-${ARCH}"
if [ "$OS" = "darwin" ] && [ "$ARCH" = "arm64" ]; then
  BINARY_NAME="nido-darwin-arm64"
fi

DOWNLOAD_URL="https://github.com/Josepavese/nido/releases/download/${LATEST_RELEASE}/${BINARY_NAME}"

echo "${CYAN}📥 Downloading ${BINARY_NAME}...${RESET}"
TMP_FILE="/tmp/nido-${LATEST_RELEASE}"

if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
  echo "${RED}❌ Download failed${RESET}"
  exit 1
fi

chmod +x "$TMP_FILE"

# Setup Nido home directory
NIDO_HOME="${HOME}/.nido"
mkdir -p "${NIDO_HOME}/bin"
mkdir -p "${NIDO_HOME}/vms"
mkdir -p "${NIDO_HOME}/run"
mkdir -p "${NIDO_HOME}/images"

# Move binary
mv "$TMP_FILE" "${NIDO_HOME}/bin/nido"

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

if [ "$OS" = "linux" ]; then
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

# --- Final Tip ---
QEMU_CMD="sudo apt update && sudo apt install qemu-system-x86 qemu-utils"
if [ "$ARCH" = "arm64" ]; then
    QEMU_CMD="sudo apt update && sudo apt install qemu-system-arm qemu-utils"
fi

echo ""
echo "${BOLD}${GREEN}🎉 Installation complete!${RESET}"
echo ""
echo "${BOLD}Next steps:${RESET}"
echo "  1. Reload shell:  ${CYAN}source ${SHELL_RC:-~/.bashrc}${RESET}"
echo "  2. Verify install: ${CYAN}nido version${RESET}"
echo "  3. Check system:   ${CYAN}nido doctor${RESET}"
echo ""
echo "${YELLOW}💡 Note: You'll need QEMU installed to run VMs${RESET}"
echo "   Linux: ${CYAN}${QEMU_CMD}${RESET}"
echo "   macOS: ${CYAN}brew install qemu${RESET}"
echo ""
echo "${BOLD}\"It's not a VM, it's a lifestyle.\" 🪺${RESET}"
