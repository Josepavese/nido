#!/usr/bin/env bash
#
# nido - Network Manager
#

# Get IP address of a VM
# Returns IP on stdout, empty string if not found
network_get_ip() {
  local vm="$1"
  local ip=""
  ip=$(virsh domifaddr "$vm" --source agent 2>/dev/null | extract_ip)
  [[ -z "$ip" ]] && ip=$(virsh domifaddr "$vm" --source lease 2>/dev/null | extract_ip)
  [[ -z "$ip" ]] && ip=$(virsh domifaddr "$vm" --source arp 2>/dev/null | extract_ip)
  if [[ -z "$ip" ]]; then
    # Fallback to ARP table lookup via MAC
    mapfile -t macs < <(virsh domiflist "$vm" 2>/dev/null | awk 'NR>2 && $0 ~ /[0-9a-fA-F]{2}:/ {print $5}')
    for mac in "${macs[@]:-}"; do
      ip=$(ip neigh show | awk -v mac="$mac" '$0 ~ mac {print $1; exit}')
      [[ -n "$ip" ]] && break
    done
  fi
  echo "$ip"
}

# Wait for IP to appear
# Returns 0 on success (prints IP to stdout), 1 on timeout
network_wait_ip() {
  local vm="$1"
  local timeout="${2:-60}"
  local start_ts end_ts ip=""

  start_ts=$(date +%s)
  
  while true; do
    ip=$(network_get_ip "$vm")
    if [[ -n "$ip" ]]; then
      echo "$ip"
      return 0
    fi
    
    end_ts=$(date +%s)
    if (( end_ts - start_ts >= timeout )); then
      return 1
    fi
    sleep 0.5
  done
}

# Get Hostname
network_get_hostname() {
  local vm="$1"
  local hn
  hn=$(virsh domhostname "$vm" 2>/dev/null || true)
  [[ -z "$hn" ]] && hn="(unknown)"
  echo "$hn"
}
