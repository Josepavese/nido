#!/usr/bin/env bash
set -euo pipefail

OS="$(uname -s)"
USER_NAME="${USER:-$(id -un)}"

# --- Colors & Icons ---
BOLD="$(tput bold || echo '')"
BLUE="$(tput setaf 4 || echo '')"
GREEN="$(tput setaf 2 || echo '')"
YELLOW="$(tput setaf 3 || echo '')"
RED="$(tput setaf 1 || echo '')"
CYAN="$(tput setaf 6 || echo '')"
RESET="$(tput sgr0 || echo '')"

INFO="  [${BLUE}i${RESET}]"
OK="  [${GREEN}ok${RESET}]"
WARN="  [${YELLOW}!${RESET}]"
ERR="  [${RED}ERR${RESET}]"

LOG_FILE="/tmp/nido-setup.log"
rm -f "$LOG_FILE"

# --- Utils ---
spinner() {
  local pid=$1
  local delay=0.1
  local spinstr='|/-\'
  while kill -0 "$pid" 2>/dev/null; do
    local temp=${spinstr#?}
    printf " [%c]" "$spinstr"
    local spinstr=$temp${spinstr%"$temp"}
    sleep $delay
    printf "\b\b\b\b"
  done
  printf "    \b\b\b\b"
}

run_task() {
    local msg=$1
    shift
    printf "  ${BOLD}${CYAN}..${RESET} %s" "$msg"
    "$@" >> "$LOG_FILE" 2>&1 &
    spinner $!
    wait $!
    if [ $? -eq 0 ]; then
        printf "\r${OK} %s\n" "$msg"
    else
        printf "\r${ERR} %s (Check ${YELLOW}$LOG_FILE${RESET})\n" "$msg"
        exit 1
    fi
}

check_sudo() {
  if ! command -v sudo >/dev/null 2>&1; then
    echo "${ERR} Error: sudo is required." >&2
    exit 1
  fi
}

setup_linux() {
  if [[ ! -r /etc/os-release ]]; then
    echo "${ERR} Error: /etc/os-release not found. Unsupported Linux distro." >&2
    exit 1
  fi
  . /etc/os-release

  echo "${INFO} Detected Linux: ${BOLD}${ID:-unknown}${RESET}"
  check_sudo

  case "${ID:-}" in
    debian|ubuntu|linuxmint|pop|kali)
      echo "${INFO} Using apt-get to gather prerequisites..."
      run_task "Updating package list" sudo apt-get update
      run_task "Installing virtualization stack" sudo apt-get install -y qemu-kvm libvirt-daemon-system libvirt-clients virtinst qemu-utils libguestfs-tools iproute2 jq
      ;;
    fedora)
      echo "${INFO} Using dnf to gather prerequisites..."
      run_task "Installing virtualization stack" sudo dnf install -u qemu-kvm libvirt virt-install guestfs-tools jq
      ;;
    *)
      if [[ "${ID_LIKE:-}" =~ "debian" ]]; then
          echo "${INFO} Derivative detected (ID_LIKE: $ID_LIKE). Using apt-get..."
          run_task "Updating package list" sudo apt-get update
          run_task "Installing virtualization stack" sudo apt-get install -y qemu-kvm libvirt-daemon-system libvirt-clients virtinst qemu-utils libguestfs-tools iproute2 jq
      else
          echo "${ERR} Unsupported distro: ${ID:-unknown}. Please install libvirt/kvm/virt-install manually." >&2
          exit 1
      fi
      ;;
  esac

  if command -v systemctl >/dev/null 2>&1 && systemctl status >/dev/null 2>&1; then
      run_task "Enabling libvirtd services" sudo systemctl enable --now libvirtd
  else
      echo "${INFO} Systemd not found/active (container?). Skipping systemctl."
  fi

  # Group membership
  local need_logout=0
  for group in libvirt kvm; do
    if getent group "$group" >/dev/null; then
      if ! id -nG "$USER_NAME" | grep -qw "$group"; then
        run_task "Adding user $USER_NAME to group $group" sudo usermod -aG "$group" "$USER_NAME"
        need_logout=1
      fi
    fi
  done

  if [[ "$need_logout" -eq 1 ]]; then
    echo "${WARN} Groups updated. You may need to logout/login for changes to take effect."
  fi
}

setup_macos() {
  echo "${INFO} Detected ${BOLD}macOS${RESET}."
  if ! command -v brew >/dev/null 2>&1; then
    echo "${ERR} Error: Homebrew is required on macOS. Install it from https://brew.sh/" >&2
    exit 1
  fi

  run_task "Updating Homebrew" brew update
  run_task "Installing dependencies" brew install bash qemu libvirt virt-manager jq

  echo "${INFO} Note: On macOS, libvirt runs as a user service or needs specialized setup."
  echo "${INFO} Services start: ${BOLD}brew services start libvirt${RESET}"
}

case "$OS" in
  Linux)
    setup_linux
    ;;
  Darwin)
    setup_macos
    ;;
  *)
    echo "${ERR} Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

echo "${OK} Nesting environment ready! 🎉"
