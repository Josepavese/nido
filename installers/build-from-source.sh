#!/usr/bin/env bash
# 🪺 Nido Source Installer - Lightweight Build from Source
# Downloads the release source archive and builds locally. No git clone required.
set -euo pipefail

# --- Colors & Icons ---
BOLD="$(tput bold 2>/dev/null || echo '')"
GREEN="$(tput setaf 2 2>/dev/null || echo '')"
CYAN="$(tput setaf 6 2>/dev/null || echo '')"
YELLOW="$(tput setaf 3 2>/dev/null || echo '')"
RED="$(tput setaf 1 2>/dev/null || echo '')"
RESET="$(tput sgr0 2>/dev/null || echo '')"

echo "${BOLD}${CYAN}"
echo "  🪺 Nido Source Build"
echo "  Lightweight build from source"
echo "${RESET}"

# Check for Go
echo "${CYAN}🔍 Checking for Go compiler...${RESET}"
if ! command -v go >/dev/null 2>&1; then
  echo "${RED}❌ Go is required to build from source${RESET}"
  echo "   Install: https://go.dev/dl/"
  exit 1
fi
echo "${GREEN}✅ Go found: $(go version)${RESET}"

# Determine latest release
echo "${CYAN}🔍 Fetching latest release...${RESET}"
LATEST_RELEASE="${NIDO_VERSION:-}"
if [ -z "$LATEST_RELEASE" ]; then
  LATEST_RELEASE=$(curl -sL https://api.github.com/repos/Josepavese/nido/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

if [ -z "$LATEST_RELEASE" ]; then
  echo "${YELLOW}⚠️  Could not fetch latest release, using 'main' branch${RESET}"
  REF_KIND="heads"
  REF_NAME="main"
else
  echo "${GREEN}✅ Latest version: ${LATEST_RELEASE}${RESET}"
  REF_KIND="tags"
  REF_NAME="$LATEST_RELEASE"
fi

# Create temporary build directory
BUILD_DIR="$(mktemp -d -t nido-build.XXXXXX)"
trap 'rm -rf "$BUILD_DIR"' EXIT
cd "$BUILD_DIR"

echo "${CYAN}📥 Downloading source archive...${RESET}"
SOURCE_URL="https://github.com/Josepavese/nido/archive/refs/${REF_KIND}/${REF_NAME}.tar.gz"
curl -fsSL "$SOURCE_URL" -o source.tar.gz
tar -xzf source.tar.gz --strip-components=1

# Build
echo ""
echo "${CYAN}🔨 Building Nido...${RESET}"
go build -o nido ./cmd/nido
go build -o nido-validator ./cmd/nido-validator

if [ ! -f "nido" ] || [ ! -f "nido-validator" ]; then
  echo "${RED}❌ Build failed${RESET}"
  exit 1
fi

echo "${GREEN}✅ Build successful!${RESET}"

# Setup Nido home directory
NIDO_HOME="${HOME}/.nido"
mkdir -p "${NIDO_HOME}/bin"
mkdir -p "${NIDO_HOME}/vms"
mkdir -p "${NIDO_HOME}/run"
mkdir -p "${NIDO_HOME}/images"
mkdir -p "${NIDO_HOME}/backups"
mkdir -p "${NIDO_HOME}/registry"

install -m 0755 nido "${NIDO_HOME}/bin/nido"
install -m 0755 nido-validator "${NIDO_HOME}/bin/nido-validator"

if [ -d registry ]; then
  cp -R registry/. "${NIDO_HOME}/registry/"
  echo "${GREEN}✅ Registry installed to ${NIDO_HOME}/registry${RESET}"
fi

# Create default config if missing
if [ ! -f "${NIDO_HOME}/config.env" ]; then
  cat > "${NIDO_HOME}/config.env" << 'EOF'
# Nido Configuration
BACKUP_DIR=${HOME}/.nido/backups
TEMPLATE_DEFAULT=template-headless
SSH_USER=vmuser
EOF
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
