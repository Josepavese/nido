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
  if ! grep -q "${NIDO_HOME}/bin" "$SHELL_RC" 2>/dev/null; then
    echo "" >> "$SHELL_RC"
    echo "# Nido VM Manager" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:${NIDO_HOME}/bin\"" >> "$SHELL_RC"
    echo "${GREEN}✅ Added to PATH in ${SHELL_RC}${RESET}"
  fi
fi

echo ""
echo "${BOLD}${GREEN}🎉 Installation complete!${RESET}"
echo ""
echo "${BOLD}Next steps:${RESET}"
echo "  1. Reload shell: ${CYAN}source ${SHELL_RC:-~/.bashrc}${RESET}"
echo "  2. Verify install: ${CYAN}nido version${RESET}"
echo "  3. Check system: ${CYAN}nido doctor${RESET}"
echo ""
echo "${YELLOW}💡 Note: You'll need QEMU installed to run VMs${RESET}"
echo "   Linux: ${CYAN}sudo apt install qemu-system-x86 qemu-utils${RESET}"
echo "   macOS: ${CYAN}brew install qemu${RESET}"
echo ""
echo "${BOLD}\"It's not a VM, it's a lifestyle.\" 🪺${RESET}"
