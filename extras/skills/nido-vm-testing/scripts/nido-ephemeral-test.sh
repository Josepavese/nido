#!/usr/bin/env bash
set -Eeuo pipefail

# Spawn a disposable Nido VM, provision it, copy the current project, run tests,
# then delete the VM. Override behavior with the NIDO_* environment variables
# documented in the usage output below.

usage() {
  cat <<'EOF'
Usage:
  extras/skills/nido-vm-testing/scripts/nido-ephemeral-test.sh

Environment:
  NIDO_BIN                 Nido binary to use. Default: nido
  NIDO_VM_NAME             VM name. Default: nido-test-<timestamp>-<pid>
  NIDO_TEST_IMAGE          Image tag. Default: ubuntu:24.04
  NIDO_MEMORY              VM memory MB. Default: 4096
  NIDO_CPUS                VM vCPUs. Default: 2
  NIDO_APP_PORT            Guest service port to forward with label "app". Default: 8080
  NIDO_PROJECT_DIR         Local project directory to upload. Default: current directory
  NIDO_REMOTE_DIR          Remote directory inside the VM. Default: /tmp/nido-app
  NIDO_PROVISION_PACKAGES  Debian/Ubuntu packages for default user-data.
  NIDO_USER_DATA           Existing cloud-init/user-data file to pass instead.
  NIDO_TEST_CMD            Test command run inside the VM project directory.
  NIDO_KEEP_VM             Set to 1 to keep the VM after the script exits.

Example:
  NIDO_TEST_CMD="make test" \
  NIDO_PROVISION_PACKAGES="ca-certificates curl git make build-essential" \
  extras/skills/nido-vm-testing/scripts/nido-ephemeral-test.sh
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
  usage
  exit 0
fi

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 127
  fi
}

need python3
need sed
need ssh
need tar

shell_quote() {
  printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

NIDO_BIN="${NIDO_BIN:-nido}"
NIDO_VM_NAME="${NIDO_VM_NAME:-nido-test-$(date +%Y%m%d%H%M%S)-$$}"
NIDO_TEST_IMAGE="${NIDO_TEST_IMAGE:-ubuntu:24.04}"
NIDO_MEMORY="${NIDO_MEMORY:-4096}"
NIDO_CPUS="${NIDO_CPUS:-2}"
NIDO_APP_PORT="${NIDO_APP_PORT:-8080}"
NIDO_PROJECT_DIR="${NIDO_PROJECT_DIR:-$PWD}"
NIDO_REMOTE_DIR="${NIDO_REMOTE_DIR:-/tmp/nido-app}"
NIDO_PROVISION_PACKAGES="${NIDO_PROVISION_PACKAGES:-ca-certificates curl git make}"
NIDO_TEST_CMD="${NIDO_TEST_CMD:-if [ -f Makefile ]; then make test; elif [ -x ./test.sh ]; then ./test.sh; else echo 'Set NIDO_TEST_CMD'; exit 2; fi}"
NIDO_KEEP_VM="${NIDO_KEEP_VM:-0}"

need "$NIDO_BIN"

if [[ ! -d "$NIDO_PROJECT_DIR" ]]; then
  echo "project directory does not exist: $NIDO_PROJECT_DIR" >&2
  exit 2
fi

VM_CREATED=0
USER_DATA_OWNED=0
USER_DATA_PATH="${NIDO_USER_DATA:-}"

cleanup() {
  status=$?
  if [[ "$NIDO_KEEP_VM" == "1" ]]; then
    echo "Keeping VM for inspection: $NIDO_VM_NAME" >&2
  elif [[ "$VM_CREATED" == "1" ]]; then
    "$NIDO_BIN" delete "$NIDO_VM_NAME" --json >/dev/null 2>&1 || true
  fi
  if [[ "$USER_DATA_OWNED" == "1" ]]; then
    rm -f "$USER_DATA_PATH"
  fi
  exit "$status"
}
trap cleanup EXIT

if [[ -z "$USER_DATA_PATH" ]]; then
  USER_DATA_PATH="$(mktemp "${TMPDIR:-/tmp}/nido-user-data.XXXXXX")"
  USER_DATA_OWNED=1
  cat >"$USER_DATA_PATH" <<EOF
#!/bin/sh
set -eux
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y $NIDO_PROVISION_PACKAGES
EOF
elif [[ ! -f "$USER_DATA_PATH" ]]; then
  echo "user-data file does not exist: $USER_DATA_PATH" >&2
  exit 2
fi

echo "Spawning $NIDO_VM_NAME from $NIDO_TEST_IMAGE"
"$NIDO_BIN" spawn "$NIDO_VM_NAME" \
  --image "$NIDO_TEST_IMAGE" \
  --memory "$NIDO_MEMORY" \
  --cpus "$NIDO_CPUS" \
  --user-data "$USER_DATA_PATH" \
  --port "app:${NIDO_APP_PORT}/tcp" \
  --json >/dev/null
VM_CREATED=1

INFO_JSON="$("$NIDO_BIN" info "$NIDO_VM_NAME" --json)"
INFO_FIELDS="$(python3 - "$INFO_JSON" <<'PY'
import json
import sys

payload = json.loads(sys.argv[1])
vm = payload["data"]["vm"]
print(vm["ssh_port"])
print(vm.get("ssh_user") or "vmuser")
host_port = ""
for item in vm.get("forwarding") or []:
    if item.get("label") == "app":
        host_port = str(item.get("host_port") or "")
        break
print(host_port)
PY
)"
SSH_PORT="$(printf '%s\n' "$INFO_FIELDS" | sed -n '1p')"
SSH_USER="$(printf '%s\n' "$INFO_FIELDS" | sed -n '2p')"
APP_HOST_PORT="$(printf '%s\n' "$INFO_FIELDS" | sed -n '3p')"

SSH_OPTS=(
  -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null
  -o ConnectTimeout=5
  -o BatchMode=yes
  -o NumberOfPasswordPrompts=0
  -p "$SSH_PORT"
)
NIDO_KEY="${HOME}/.nido/nido_ed25519"
if [[ -f "$NIDO_KEY" ]]; then
  SSH_OPTS=(-i "$NIDO_KEY" "${SSH_OPTS[@]}")
fi
SSH_TARGET="${SSH_USER}@127.0.0.1"
REMOTE_DIR_Q="$(shell_quote "$NIDO_REMOTE_DIR")"

echo "Waiting for SSH on 127.0.0.1:$SSH_PORT"
for ((attempt = 1; attempt <= 90; attempt++)); do
  if ssh "${SSH_OPTS[@]}" "$SSH_TARGET" "echo ready" >/dev/null 2>&1; then
    break
  fi
  sleep 5
done
ssh "${SSH_OPTS[@]}" "$SSH_TARGET" "echo ready" >/dev/null

echo "Waiting for guest provisioning to finish"
ssh "${SSH_OPTS[@]}" "$SSH_TARGET" "if command -v cloud-init >/dev/null 2>&1; then if command -v timeout >/dev/null 2>&1; then sudo timeout 1800 cloud-init status --wait; else sudo cloud-init status --wait; fi; fi"

echo "Uploading project to $NIDO_REMOTE_DIR"
ssh "${SSH_OPTS[@]}" "$SSH_TARGET" "rm -rf -- $REMOTE_DIR_Q && mkdir -p -- $REMOTE_DIR_Q"
tar \
  --exclude .git \
  --exclude .nido \
  --exclude node_modules \
  --exclude dist \
  --exclude build \
  -C "$NIDO_PROJECT_DIR" \
  -cf - . | ssh "${SSH_OPTS[@]}" "$SSH_TARGET" "tar -xf - -C $REMOTE_DIR_Q"

if [[ -n "$APP_HOST_PORT" ]]; then
  echo "Forwarded app port: http://127.0.0.1:$APP_HOST_PORT"
fi

echo "Running test command: $NIDO_TEST_CMD"
ssh "${SSH_OPTS[@]}" "$SSH_TARGET" "cd $REMOTE_DIR_Q && $NIDO_TEST_CMD"
