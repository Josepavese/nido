# Nido MCP Server Logic (JSON-RPC 2.0 Loop)

# --- Protocol Handling ---

send_response() {
    local id="$1"
    local result="$2"
    # Write ONLY to FD 3 (Real Stdout)
    jq -n --argjson id "$id" --argjson result "$result" '{jsonrpc: "2.0", id: $id, result: $result}' >&3
}

send_error() {
    local id="$1"
    local code="$2"
    local msg="$3"
    # Write ONLY to FD 3 (Real Stdout)
    jq -n --argjson id "$id" --argjson code "$code" --arg msg "$msg" \
        '{jsonrpc: "2.0", id: $id, error: {code: $code, message: $msg}}' >&3
}

# --- Utils ---
sanitize_input() {
    local input="$1"
    # Allow alphanumeric, dashes, underscores, dots.
    # Reject anything else to prevent traversal or injection.
    if [[ ! "$input" =~ ^[a-zA-Z0-9_.-]+$ ]]; then
        return 1
    fi
    # Also explicitly block ".."
    if [[ "$input" == *".."* ]]; then
        return 1
    fi
    return 0
}

# --- Handler ---
handle_tool_call() {
    local id="$1"
    local tool_name="$2"
    local args="$3"
    
    case "$tool_name" in
        vm_list)
            local vms_json
            vms_json=$(json_vm_list)
            local content
            content=$(jq -n --argjson vms "$vms_json" '{content: [{type: "text", text: ($vms | tostring)}] }')
            send_response "$id" "$content"
            ;;
            
        vm_info)
            local name
            name=$(echo "$args" | jq -r '.name // empty')
            if ! sanitize_input "$name"; then
                 send_error "$id" -32602 "Invalid parameter 'name': must be alphanumeric/dashes/dots"
                 return
            fi

            local info_json
            info_json=$(json_vm_info "$name")
            local content
            content=$(jq -n --argjson info "$info_json" '{content: [{type: "text", text: ($info | tostring)}] }')
            send_response "$id" "$content"
            ;;
            
        vm_start)
            local name
            name=$(echo "$args" | jq -r '.name // empty')
            if ! sanitize_input "$name"; then
                 send_error "$id" -32602 "Invalid parameter 'name': must be alphanumeric/dashes/dots"
                 return
            fi
            
            if vm_start "$name"; then
                 local ip
                 if ip=$(vm_wait_ip "$name" 5); then
                     local msg="VM $name started. IP: $ip"
                     local content
                     content=$(jq -n --arg msg "$msg" '{content: [{type: "text", text: $msg}] }')
                     send_response "$id" "$content"
                 else
                     local msg="VM $name started (IP lookup timed out, check later)."
                     local content
                     content=$(jq -n --arg msg "$msg" '{content: [{type: "text", text: $msg}] }')
                     send_response "$id" "$content"
                 fi
            else
                 send_error "$id" -32000 "Failed to start VM $name"
            fi
            ;;
            
        vm_stop)
            local name
            name=$(echo "$args" | jq -r '.name // empty')
            if ! sanitize_input "$name"; then
                 send_error "$id" -32602 "Invalid parameter 'name': must be alphanumeric/dashes/dots"
                 return
            fi

            if vm_stop "$name"; then
                 local content
                 content=$(jq -n --arg msg "VM $name stopped." '{content: [{type: "text", text: $msg}] }')
                 send_response "$id" "$content"
            else
                 send_error "$id" -32000 "Failed to stop VM $name"
            fi
            ;;
            
        vm_delete)
            local name
            name=$(echo "$args" | jq -r '.name // empty')
            if ! sanitize_input "$name"; then
                 send_error "$id" -32602 "Invalid parameter 'name': must be alphanumeric/dashes/dots"
                 return
            fi

            # Double check it exists before attempting destroy
            if ! virsh dominfo "$name" >/dev/null 2>&1; then
                 send_error "$id" -32000 "VM not found: $name"
                 return
            fi

            if vm_destroy "$name" "$VMS_POOL"; then
                 # Let's interact with pool to remove volume
                 local vol_name="${name}.qcow2"
                 if virsh vol-info --pool "$VMS_POOL" "$vol_name" >/dev/null 2>&1; then
                    virsh vol-delete --pool "$VMS_POOL" "$vol_name" >/dev/null 2>&1 || true
                 fi
                 
                 local content
                 content=$(jq -n --arg msg "VM $name deleted (including disk)." '{content: [{type: "text", text: $msg}] }')
                 send_response "$id" "$content"
            else
                 send_error "$id" -32000 "Failed to delete VM $name"
            fi
            ;;
        
        vm_create)
             # Logic for creation
             local name template
             name=$(echo "$args" | jq -r '.name // empty')
             if ! sanitize_input "$name"; then
                 send_error "$id" -32602 "Invalid parameter 'name': must be alphanumeric/dashes/dots"
                 return
             fi

             template=$(echo "$args" | jq -r '.template // empty')
             # If template is provided, validate it too
             if [[ -n "$template" ]] && ! sanitize_input "$template"; then
                 send_error "$id" -32602 "Invalid parameter 'template': must be alphanumeric/dashes/dots"
                 return
             fi
             
             [[ -z "$template" ]] && template="$TEMPLATE_DEFAULT"
             
             local backup="$BACKUP_DIR/${template}.compact.qcow2"
             local ret out
             
             # Use || true to prevent set -e from crashing the script on failure
             out=$(vm_create "$name" "$template" "$POOL_PATH" "$backup" \
                  "$VM_MEM_MB" "$VM_VCPUS" "$VM_OS_VARIANT" \
                  "$NETWORK_HOSTONLY" "$NETWORK_NAT" \
                  "$GRAPHICS" "$VM_NESTED" || true)
             # However, we need the exit code. 
             # vm_create prints to stdout on success (PID) and stderr on failure.
             # If it fails, out might be empty or contain partial output.
             # To robustly check success, we check if PID exists or if out is a valid PID.
             
             if [[ "$out" =~ ^[0-9]+$ ]]; then
                 ret=0
             else
                 ret=1
             fi
             
             if [[ $ret -eq 0 ]]; then
                 local pid="$out"
                 while kill -0 "$pid" 2>/dev/null; do
                    sleep 0.2
                 done
                 
                 local content
                 content=$(jq -n --arg msg "VM $name created successfully." '{content: [{type: "text", text: $msg}] }')
                 send_response "$id" "$content"
             else
                 send_error "$id" -32000 "Failed to create VM $name"
             fi
             ;;

        *)
            send_error "$id" -32601 "Tool not found: $tool_name"
            ;;
    esac
}

run_mcp_server() {
  # Main Loop
  while IFS= read -r line; do
    local method id params
    method=$(echo "$line" | jq -r '.method // empty')
    id=$(echo "$line" | jq -r '.id // null')
    
    if [[ -z "$method" ]]; then continue; fi
    
    case "$method" in
      initialize)
        cat <<EOF >&3
{
  "jsonrpc": "2.0",
  "id": $id,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": { "tools": {} },
    "serverInfo": { "name": "nido-mcp", "version": "1.0.0" }
  }
}
EOF
        ;;
        
      notifications/initialized)
        # No response needed
        ;;
        
      tools/list)
        # Dynamic tool list
        cat <<EOF >&3
{
  "jsonrpc": "2.0",
  "id": $id,
  "result": {
    "tools": [
      {
        "name": "vm_list",
        "description": "List all configured virtual machines. Returns specific status (running/stopped) and IP addresses.",
        "inputSchema": { "type": "object", "properties": {} }
      },
      {
        "name": "vm_info",
        "description": "Get detailed technical specifications for a VM (CPU, RAM, MAC, State).",
        "inputSchema": {
          "type": "object",
          "properties": { "name": { "type": "string", "description": "The exact name of the VM" } },
          "required": ["name"]
        }
      },
      {
        "name": "vm_start",
        "description": "Power on a virtual machine. This tool waits up to 5s for an IP address to be assigned.",
        "inputSchema": {
          "type": "object",
          "properties": { "name": { "type": "string", "description": "The exact name of the VM" } },
          "required": ["name"]
        }
      },
      {
         "name": "vm_stop",
         "description": "Gracefully shutdown a virtual machine (ACPI signal).",
         "inputSchema": {
           "type": "object",
           "properties": { "name": { "type": "string", "description": "The exact name of the VM" } },
           "required": ["name"]
         }
      },
      {
         "name": "vm_delete",
         "description": "Permanently destroy a VM and remove its associated disk image. This is destructive.",
         "inputSchema": {
           "type": "object",
           "properties": { "name": { "type": "string", "description": "The exact name of the VM" } },
           "required": ["name"]
         }
      },
      {
         "name": "vm_create",
         "description": "Create a new VM from a template. This clones the disk and defines the domain.",
         "inputSchema": {
           "type": "object",
           "properties": { 
               "name": { "type": "string", "description": "Name for the new VM (alphanumeric)" },
               "template": { "type": "string", "description": "Template to use (e.g. template-headless)" }
           },
           "required": ["name"]
         }
      }
    ]
  }
}
EOF
        ;;
        
      tools/call)
         # params comes from stdin which is fine, but output via handle_tool_call uses send_response (> &3)
         params=$(echo "$line" | jq -c '.params')
         tool_name=$(echo "$params" | jq -r '.name')
         args=$(echo "$params" | jq -c '.arguments')
         handle_tool_call "$id" "$tool_name" "$args"
         ;;
         
      ping)
        send_response "$id" "{}"
        ;;
        
      *)
        if [[ "$id" != "null" ]]; then
             send_error "$id" -32601 "Method not found: $method"
        fi
        ;;
    esac
  done
}
