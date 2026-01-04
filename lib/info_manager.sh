#!/usr/bin/env bash
#
# nido - Info Manager (JSON Output)
#

# Returns JSON array of all VMs
vm_list_json() {
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
    
    # We don't fetch IP here to be fast. IP fetching is slow.
    # MCP client should call vm_info/vm_list_detailed if they need IPs.
    
    json="$json{\"name\": \"$vm\", \"state\": \"$state\"}"
  done
  
  json="$json]"
  echo "$json"
}

# Returns detailed JSON object for a VM
vm_info_json() {
  local name="$1"
  
  if ! virsh dominfo "$name" >/dev/null 2>&1; then
    echo "{}" 
    return 1
  fi
  
  local hn ip state cpu mem maxmem
  hn=$(vm_hostname "$name")
  ip=$(vm_get_ip "$name")
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
  "ssh_user": "$SSH_USER",
  "memory_kb": "${mem:-0}",
  "cpu_count": "${cpu:-0}"
}
EOF
}
