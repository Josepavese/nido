#!/usr/bin/env bash
#
# nido installer
# Usage: curl -fsSL https://.../get-nido.sh | bash
#

set -euo pipefail

REPO_URL="https://github.com/your-org/nido" # Replace with actual URL
INSTALL_DIR="${HOME}/.nido"
BIN_DIR="${INSTALL_DIR}/bin"

echo "🐣 nido installer"

# 1. Prerequisites check
if ! command -v git >/dev/null 2>&1; then
  echo "Error: git is required to install." >&2
  exit 1
fi

# 2. Checkout/Update
if [[ -d "$INSTALL_DIR" ]]; then
  echo "Updating existing installation in $INSTALL_DIR..."
  cd "$INSTALL_DIR"
  git pull
else
  echo "Cloning nido to $INSTALL_DIR..."
  git clone "$REPO_URL" "$INSTALL_DIR"
fi

# 3. Dependencies
echo "Running dependency setup..."
bash "$BIN_DIR/setup_deps.sh"

# 4. Path setup
SHELL_RC=""
case "$SHELL" in
  */bash) SHELL_RC="$HOME/.bashrc" ;;
  */zsh)  SHELL_RC="$HOME/.zshrc" ;;
esac

if [[ -n "$SHELL_RC" ]]; then
  if ! grep -q "$BIN_DIR" "$SHELL_RC"; then
    echo "Adding $BIN_DIR to PATH in $SHELL_RC..."
    echo "" >> "$SHELL_RC"
    echo "# nido" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:$BIN_DIR\"" >> "$SHELL_RC"
    echo "Run 'source $SHELL_RC' to update your current shell."
  else
    echo "Path already configured in $SHELL_RC."
  fi
else
  echo "Could not detect shell configuration file. Please add this to your PATH manually:"
  echo "  export PATH=\"\$PATH:$BIN_DIR\""
fi

echo "✅ nido installed successfully!"
echo "Try running: nido --help"
