#!/usr/bin/env bash
#
# nido - Output Manager (JSON generators)
#

# Requires: network_manager.sh

# Returns JSON array of all VMs
json_vm_list() {
  local regex="${1:-.}"
  # Ensure we have an empty array if no VMs
  local json="["
  
  # Get list of domains
  mapfile -t vms < <(virsh list --all --name 2>/dev/null | awk 'NF' | grep -E "$regex" || true)
  
  local first=true
  for vm in "${vms[@]}"; do
    if [[ "$first" == "true" ]]; then
      first=false
    else
      json="$json,"
    fi
    
    local state
    state=$(virsh domstate "$vm" 2>/dev/null || echo "unknown")
    
    json="$json{\"name\": \"$vm\", \"state\": \"$state\"}"
  done
  
  json="$json]"
  echo "$json"
}

# Returns detailed JSON object for a VM
json_vm_info() {
  local name="$1"
  
  if ! virsh dominfo "$name" >/dev/null 2>&1; then
    echo "{}" 
    return 1
  fi
  
  local hn ip state cpu mem
  hn=$(network_get_hostname "$name")
  ip=$(network_get_ip "$name")
  state=$(virsh domstate "$name" 2>/dev/null || echo "unknown")
  
  # Get basic stats
  mem=$(virsh dominfo "$name" | grep "Used memory" | awk '{print $3}')
  cpu=$(virsh dominfo "$name" | grep "CPU(s)" | awk '{print $2}')
  
  [[ -z "$ip" ]] && ip=""

  cat <<EOF
{
  "name": "$name",
  "hostname": "$hn",
  "state": "$state",
  "ip": "$ip",
  "ssh_user": "${SSH_USER:-vmuser}",
  "memory_kb": "${mem:-0}",
  "cpu_count": "${cpu:-0}"
}
EOF
}
