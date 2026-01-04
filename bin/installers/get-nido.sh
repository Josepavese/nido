#!/usr/bin/env bash
#
# nido installer
# Usage: curl -fsSL https://.../get-nido.sh | bash
#

set -euo pipefail

REPO_URL="https://github.com/Josepavese/nido"
INSTALL_DIR="${HOME}/.nido"
BIN_DIR="${INSTALL_DIR}/bin"

echo "🐣 nido installer: making nesting easy since 2026"

# 1. Prerequisites check
if ! command -v git >/dev/null 2>&1; then
  echo "❌ Error: git is missing. Even birds need wings to fly!" >&2
  exit 1
fi

# 2. Checkout/Update
if [[ -d "$INSTALL_DIR" ]]; then
  echo "🔄 Nest found! Tidying up $INSTALL_DIR..."
  cd "$INSTALL_DIR"
  git pull
else
  echo "🏗️ Building a new nest in $INSTALL_DIR..."
  git clone "$REPO_URL" "$INSTALL_DIR"
fi

# 3. Dependencies
echo "🐛 Hunting for dependencies (this might require a password for sudo)..."
bash "$BIN_DIR/setup_deps.sh"

# 4. Path setup
SHELL_RC=""
case "$SHELL" in
  */bash) SHELL_RC="$HOME/.bashrc" ;;
  */zsh)  SHELL_RC="$HOME/.zshrc" ;;
esac

if [[ -n "$SHELL_RC" ]]; then
  if ! grep -q "$BIN_DIR" "$SHELL_RC"; then
    echo "🗺️ Marking the path to the nest in $SHELL_RC..."
    echo "" >> "$SHELL_RC"
    echo "# nido: where the VMs sleep" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:$BIN_DIR\"" >> "$SHELL_RC"
    echo "✨ Magic happened. Run 'source $SHELL_RC' to wake up the path."
  else
    echo "✅ The path is already clear. No more marking needed."
  fi
else
  echo "🤷 Could not find your shell's diary (.bashrc/.zshrc). Add this manually if you want nido to work:"
  echo "  export PATH=\"\$PATH:$BIN_DIR\""
fi

echo "🎉 nido is ready to hatch!"
echo "Try running: nido --help (it won't bite, promise)"
