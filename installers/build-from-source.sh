#!/usr/bin/env bash
# 🪺 Nido Source Installer - Lightweight Build from Source
# Downloads only essential source files and builds locally.
# Perfect for users who want the latest code without cloning the entire repo.
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
LATEST_RELEASE=$(curl -sL https://api.github.com/repos/Josepavese/nido/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
  echo "${YELLOW}⚠️  Could not fetch latest release, using 'main' branch${RESET}"
  BRANCH="main"
else
  echo "${GREEN}✅ Latest version: ${LATEST_RELEASE}${RESET}"
  BRANCH="$LATEST_RELEASE"
fi

# Create temporary build directory
BUILD_DIR="$(mktemp -d -t nido-build.XXXXXX)"
cd "$BUILD_DIR"

echo "${CYAN}📥 Downloading source files...${RESET}"

# Base URL for raw files
BASE_URL="https://raw.githubusercontent.com/Josepavese/nido/${BRANCH}"

# Download essential files
curl -fsSL "${BASE_URL}/go.mod" -o go.mod
curl -fsSL "${BASE_URL}/go.sum" -o go.sum

# Download cmd/ directory structure
echo "${CYAN}📦 Downloading cmd/...${RESET}"
mkdir -p cmd/nido
curl -fsSL "${BASE_URL}/cmd/nido/main.go" -o cmd/nido/main.go

# Download internal/ directory (this is the tricky part - we need to get the file list)
echo "${CYAN}📦 Downloading internal/...${RESET}"

# Essential internal packages
PACKAGES=(
  "config"
  "image"
  "mcp"
  "provider"
  "registry/builder"
  "ui"
)

for pkg in "${PACKAGES[@]}"; do
  mkdir -p "internal/$pkg"
  # Fetch directory listing from GitHub API
  FILES=$(curl -sL "https://api.github.com/repos/Josepavese/nido/contents/internal/$pkg?ref=${BRANCH}" | grep '"name"' | grep '\.go"' | sed -E 's/.*"([^"]+\.go)".*/\1/')
  
  for file in $FILES; do
    echo "  → internal/$pkg/$file"
    curl -fsSL "${BASE_URL}/internal/$pkg/$file" -o "internal/$pkg/$file"
  done
done

# Build
echo ""
echo "${CYAN}🔨 Building Nido...${RESET}"
go build -o nido ./cmd/nido

if [ ! -f "nido" ]; then
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

# Move binary
mv nido "${NIDO_HOME}/bin/nido"

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

# Cleanup
cd /
rm -rf "$BUILD_DIR"

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
