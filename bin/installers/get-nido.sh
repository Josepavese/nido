#!/usr/bin/env bash
#
# 🪺 nido installer: The path to the perfect nest
# Usage: curl -fsSL https://.../get-nido.sh | bash
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

# --- Header ---
clear
cat << "EOF"
    _   _ ___ ____   ___  
   | \ | |_ _|  _ \ / _ \ 
   |  \| || || | | | | | |
   | |\  || || |_| | |_| |
   |_| \_|___|____/ \___/ 
                          
EOF
echo "${BOLD}${BLUE}  Where your local VMs come to life.${RESET}"
echo "  ------------------------------------"
echo ""

REPO_URL="https://github.com/Josepavese/nido"
INSTALL_DIR="${HOME}/.nido"
BIN_DIR="${INSTALL_DIR}/bin"

# 1. Prerequisites check
echo "${STEP} ${BOLD}[1/4] Checking prerequisites...${RESET}"
if ! command -v git >/dev/null 2>&1; then
  echo "  ${ERR} Error: git is missing. Even birds need wings to fly!" >&2
  exit 1
fi
echo "  ${OK} git found. Preparing for takeoff."

# 2. Checkout/Update
echo ""
echo "${STEP} ${BOLD}[2/4] Building the nest...${RESET}"
if [[ -d "$INSTALL_DIR" ]]; then
  echo "  ${INFO} Nest found at ${YELLOW}$INSTALL_DIR${RESET}. Tidying up..."
  cd "$INSTALL_DIR"
  git pull --quiet
else
  echo "  ${INFO} Creating a cozy spot in ${YELLOW}$INSTALL_DIR${RESET}..."
  git clone --quiet "$REPO_URL" "$INSTALL_DIR"
fi
echo "  ${OK} Repository ready. The nest is structured."

# 3. Dependencies
echo ""
echo "${STEP} ${BOLD}[3/4] Hunting for dependencies...${RESET}"
echo "  ${INFO} This might require a password for ${BOLD}sudo${RESET} (the worms are deep)."
bash "$BIN_DIR/setup_deps.sh"

# 4. Path setup
echo ""
echo "${STEP} ${BOLD}[4/4] Marking the path...${RESET}"
SHELL_RC=""
case "$SHELL" in
  */bash) SHELL_RC="$HOME/.bashrc" ;;
  */zsh)  SHELL_RC="$HOME/.zshrc" ;;
esac

if [[ -n "$SHELL_RC" ]]; then
  if ! grep -q "$BIN_DIR" "$SHELL_RC"; then
    echo "  ${INFO} Adding the nest to your shell's diary (${YELLOW}$SHELL_RC${RESET})..."
    echo "" >> "$SHELL_RC"
    echo "# nido: where the VMs sleep" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:$BIN_DIR\"" >> "$SHELL_RC"
    echo "  ${OK} Path marked. Your shell now knows the way."
  else
    echo "  ${OK} The path is already clear in your shell's diary."
  fi
else
  echo "  ${WARN} Could not find your shell's diary (.bashrc/.zshrc)."
  echo "  Please add this manually to your PATH:"
  echo "  ${BOLD}export PATH=\"\$PATH:$BIN_DIR\"${RESET}"
fi

# Conclusion
echo ""
echo "${BOLD}${GREEN}  CONGRATULATIONS! ${BIRD}${RESET}"
echo "  Your nido is ready. Your VMs are waiting to hatch."
echo ""
echo "  ${BOLD}Quick Start:${RESET}"
echo "  1. Restart your terminal or run: ${CYAN}source ${SHELL_RC:-~/.bashrc}${RESET}"
echo "  2. Spawn your first VM: ${CYAN}nido spawn my-test-vm${RESET}"
echo ""
echo "  ${BLUE}\"A bird doesn't sing because it has an answer, it sings because it has a VM.\"${RESET}"
echo "  - (Probably) a very tech-savvy bird"
echo ""
