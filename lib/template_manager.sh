#!/usr/bin/env bash
#
# nido - Template Manager
#

# Create a compressed template from a source VM
# Arguments: source_vm, template_name, backup_dir, vms_pool
# Returns: 0 on success, 1 on error
template_create() {
  local src="$1"
  local tpl="$2"
  local backup_dir="$3"
  local vms_pool="$4"
  
  local template_name="$tpl"
  local compressed_path="$backup_dir/${template_name}.compact.qcow2"
  local disk_path
  local tmpdir tmpdisk

  if ! virsh dominfo "$src" >/dev/null 2>&1; then
    echo "Source VM not found: $src" >&2
    return 1
  fi

  if virsh domstate "$src" | grep -q "running"; then
    echo "Source VM is running; shut it down before creating template: $src" >&2
    return 1
  fi

  disk_path=$(virsh domblklist --details "$src" | awk '$2 == "disk" && $3 == "vda" {print $4; exit}')
  if [[ -z "$disk_path" || ! -f "$disk_path" ]]; then
    echo "Source disk not found for VM: $src" >&2
    return 1
  fi

  mkdir -p "$backup_dir"

  if [[ -e "$compressed_path" ]]; then
    echo "Compressed template already exists: $compressed_path" >&2
    return 1
  fi

  tmpdir=$(mktemp -d)
  tmpdisk="$tmpdir/source.qcow2"
  
  # Check if volume is in pool or just a file
  local vol_name
  vol_name=$(basename "$disk_path")
  
  if virsh vol-info --pool "$vms_pool" "$vol_name" >/dev/null 2>&1; then
    virsh vol-download --pool "$vms_pool" "$vol_name" "$tmpdisk" >/dev/null 2>&1
  else
    cp "$disk_path" "$tmpdisk"
  fi

  # Background compression
  qemu-img convert -O qcow2 -c "$tmpdisk" "$compressed_path" &
  local pid=$!
  
  # Note: The caller is responsible for the spinner/waiting.
  # We return the PID so the caller can wait on it.
  echo "$pid"
  
  # We need a way to cleanup tmpdir after the background process finishes.
  # Since we return immediately with PID, we can't delete it here.
  # SOLUTION: We will run the cleanup in a subshell that waits for the PID.
  (
      wait $pid
      rm -rf "$tmpdir"
  ) & disown
  
  return 0
}
