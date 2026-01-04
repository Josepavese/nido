#!/usr/bin/env bash
#
# nido - VM Manager Core Logic
#

# Create a VM (disk cloning + definition)
# Returns 0 on success, 1 on error
vm_create() {
  local name="$1"
  local template="$2"
  local pool_path="$3"
  local backup_path="$4"
  local memory="$5"
  local vcpus="$6"
  local os_variant="$7"
  local net_hostonly="$8"
  local net_nat="$9"
  local graphics="${10}"
  local nested="${11}"

  local disk_path="$pool_path/vms/${name}.qcow2"

  if virsh dominfo "$name" >/dev/null 2>&1; then
    echo "Domain already exists: $name" >&2
    return 1 # ALREADY_EXISTS
  fi

  if [[ ! -f "$backup_path" ]]; then
    echo "Compressed template not found: $backup_path" >&2
    return 1 # TEMPLATE_NOT_FOUND
  fi

  if [[ -e "$disk_path" ]]; then
    echo "Disk already exists: $disk_path" >&2
    return 1 # DISK_EXISTS
  fi

  mkdir -p "$pool_path/vms"
  
  # Background clone (for spinner handling by caller)
  qemu-img convert -O qcow2 "$backup_path" "$disk_path" &
  local pid=$!
  echo "$pid" # Return PID for spinner tracking
  
  # We assume caller waits for PID. 
  # But we can't verify exit code if we just return PID logic-wise here.
  # The CLI wrapper will handle the wait.
  return 0
}

vm_start() {
  local name="$1"
  if ! virsh dominfo "$name" >/dev/null 2>&1; then
    return 2 # NOT_FOUND
  fi
  if virsh domstate "$name" | grep -q "running"; then
    return 0 # ALREADY_RUNNING
  fi
  virsh start "$name" >/dev/null
}

vm_stop() {
  local name="$1"
  if ! virsh dominfo "$name" >/dev/null 2>&1; then
    return 2 # NOT_FOUND
  fi
  if virsh domstate "$name" | grep -q "running"; then
    virsh shutdown "$name" >/dev/null
  fi
}

vm_destroy() {
  local name="$1"
  local vms_pool="$2"
  
  if ! virsh dominfo "$name" >/dev/null 2>&1; then
    return 2 # NOT_FOUND
  fi

  if virsh domstate "$name" | grep -q "running"; then
    virsh destroy "$name" >/dev/null 2>&1
  fi

  local disk_path
  disk_path=$(virsh domblklist --details "$name" | awk '$2 == "disk" && $3 == "vda" {print $4; exit}')
  
  virsh undefine "$name" --nvram >/dev/null 2>&1 || virsh undefine "$name" >/dev/null 2>&1

  if [[ -n "$disk_path" ]]; then
    local vol_name
    vol_name=$(basename "$disk_path")
    if virsh vol-info --pool "$vms_pool" "$vol_name" >/dev/null 2>&1; then
      virsh vol-delete --pool "$vms_pool" "$vol_name" >/dev/null 2>&1
    fi
  fi
}
