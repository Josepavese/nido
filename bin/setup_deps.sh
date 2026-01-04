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
    debian|ubuntu|linuxmint|pop)
      echo "Using apt-get..."
      sudo apt-get update
      sudo apt-get install -y qemu-kvm libvirt-daemon-system libvirt-clients virtinst qemu-utils libguestfs-tools iproute2 jq
      ;;
    fedora)
      echo "Using dnf..."
      sudo dnf install -y qemu-kvm libvirt virt-install guestfs-tools jq
      ;;
    *)
      echo "Unsupported distro: ${ID:-unknown}. Please install libvirt/kvm/virt-install manually." >&2
      exit 1
      ;;
  esac

  echo "Enabling libvirtd..."
  sudo systemctl enable --now libvirtd

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

if ! command -v sudo >/dev/null 2>&1; then
  echo "sudo is required. Install it and re-run this script." >&2
  exit 1
fi

if [[ ! -r /etc/os-release ]]; then
  echo "/etc/os-release not found; unsupported distro." >&2
  exit 1
fi

. /etc/os-release

case "${ID:-}" in
  debian|ubuntu|linuxmint)
    PKG_MGR="apt-get"
    ;;
  *)
    echo "Unsupported distro: ${ID:-unknown}. Install libvirt/qemu/virtinst manually." >&2
    exit 1
    ;;
esac

REQUIRED_PKGS=(
  qemu-kvm
  libvirt-daemon-system
  libvirt-clients
  virtinst
  qemu-utils
  libguestfs-tools
  iproute2
)

echo "Installing prerequisites (idempotent) ..."
sudo "$PKG_MGR" update -y
sudo "$PKG_MGR" install -y "${REQUIRED_PKGS[@]}"

echo "Enabling libvirt service ..."
sudo systemctl enable --now libvirtd

USER_NAME="${USER:-$(id -un)}"
NEED_LOGOUT=0

ensure_group() {
  local group="$1"
  if ! id -nG "$USER_NAME" | grep -qw "$group"; then
    sudo usermod -aG "$group" "$USER_NAME"
    NEED_LOGOUT=1
  fi
}

ensure_group "libvirt"
ensure_group "kvm"

if [[ "$NEED_LOGOUT" -eq 1 ]]; then
  echo "Added $USER_NAME to libvirt/kvm. Logout/login required."
else
  echo "User groups already ok: libvirt/kvm."
fi

echo "Sanity check: virsh list --all"
if ! virsh list --all >/dev/null 2>&1; then
  echo "virsh access failed. Check group membership or libvirt socket." >&2
  exit 1
fi

echo "Done."
