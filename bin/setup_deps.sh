#!/usr/bin/env bash
set -euo pipefail

OS="$(uname -s)"
USER_NAME="${USER:-$(id -un)}"

check_sudo() {
  if ! command -v sudo >/dev/null 2>&1; then
    echo "Error: sudo is required." >&2
    exit 1
  fi
}

setup_linux() {
  if [[ ! -r /etc/os-release ]]; then
    echo "Error: /etc/os-release not found. Unsupported Linux distro." >&2
    exit 1
  fi
  . /etc/os-release

  echo "Detected Linux: ${ID:-unknown}"
  check_sudo

  case "${ID:-}" in
    debian|ubuntu|linuxmint|pop|kali)
      echo "Using apt-get..."
      sudo apt-get update
      sudo apt-get install -y qemu-kvm libvirt-daemon-system libvirt-clients virtinst qemu-utils libguestfs-tools iproute2 jq
      ;;
    fedora)
      echo "Using dnf..."
      sudo dnf install -y qemu-kvm libvirt virt-install guestfs-tools jq
      ;;
    *)
      # Fallback to ID_LIKE for derivatives
      if [[ "${ID_LIKE:-}" =~ "debian" ]]; then
          echo "Derivative detected (ID_LIKE: $ID_LIKE). Using apt-get..."
          sudo apt-get update
          sudo apt-get install -y qemu-kvm libvirt-daemon-system libvirt-clients virtinst qemu-utils libguestfs-tools iproute2 jq
      else
          echo "Unsupported distro: ${ID:-unknown}. Please install libvirt/kvm/virt-install manually." >&2
          exit 1
      fi
      ;;
  esac

  echo "Enabling libvirtd..."
  if command -v systemctl >/dev/null 2>&1 && systemctl status >/dev/null 2>&1; then
      sudo systemctl enable --now libvirtd
  else
      echo "Systemd not found/active (container?). Skipping systemctl."
  fi

  # Group membership
  local need_logout=0
  for group in libvirt kvm; do
    if getent group "$group" >/dev/null; then
      if ! id -nG "$USER_NAME" | grep -qw "$group"; then
        echo "Adding user $USER_NAME to group $group..."
        sudo usermod -aG "$group" "$USER_NAME"
        need_logout=1
      fi
    fi
  done

  if [[ "$need_logout" -eq 1 ]]; then
    echo "Groups updated. You may need to logout/login for changes to take effect."
  fi
}

setup_macos() {
  echo "Detected macOS."
  if ! command -v brew >/dev/null 2>&1; then
    echo "Error: Homebrew is required on macOS." >&2
    echo "Install it from https://brew.sh/" >&2
    exit 1
  fi

  echo "Updating Homebrew..."
  brew update

  echo "Installing dependencies..."
  brew install bash qemu libvirt virt-manager jq

  echo "Note: On macOS, libvirt runs as a user service or needs specialized setup."
  echo "You verified dependencies are installed. Ensure your libvirt backend (e.g., qemu) is configured."
  echo "Services start:"
  echo "brew services start libvirt"
}

case "$OS" in
  Linux)
    setup_linux
    ;;
  Darwin)
    setup_macos
    ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

echo "Setup complete. Run 'nido selftest' to verify."
