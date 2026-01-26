#!/bin/sh
# nido-init: Minimal & Robust Guest Initialization Agent
# Purpose: Configure guest from Kernel Command Line in minimal environments.
# Compatibility: POSIX sh, BusyBox, Alpine, Wolfi, Debian.
#
# Robustness Features:
# - set -eu (Exit on error, uninitialized vars)
# - Atomic locking via mkdir
# - Idempotent operations
# - Fallback logging to /dev/kmsg

set -eu

# Configuration
LOCK_DIR="/run/nido-init.lock"
CMDLINE_FILE="/proc/cmdline"
LOG_TAG="nido-init"

# -----------------------------------------------------------------------------
# Logging & Utilities
# -----------------------------------------------------------------------------

log() {
    local msg="$1"
    # Try logger, fallback to echo with timestamp
    if command -v logger >/dev/null 2>&1; then
        logger -t "$LOG_TAG" -s "$msg" 2>/dev/null || echo "$(date -Iseconds) [$LOG_TAG] $msg"
    else
        echo "$(date 2>/dev/null || echo "INIT") [$LOG_TAG] $msg"
    fi
    
    # Optional: Write to Kernel Ringbuffer implementation if writable
    if [ -w /dev/kmsg ]; then
        echo "<6>[$LOG_TAG] $msg" > /dev/kmsg 2>/dev/null || true
    fi
}

error() {
    log "ERROR: $1"
    exit 1
}

has_cmd() {
    command -v "$1" >/dev/null 2>&1
}

# -----------------------------------------------------------------------------
# Parsing Logic (Robust)
# -----------------------------------------------------------------------------

# Usage: get_param "key"
# Returns value of nido.key=... from /proc/cmdline
# Handles quotes: nido.key="val" -> val
get_param() {
    local key="nido.$1"
    local val=""
    
    # 1. Read cmdline
    # 2. Iterate tokens (space separated)
    # 3. Match key= prefix
    # 4. Strip quotes
    # Note: We use 'tr' to split spaces to newlines for simple iteration in pure sh.
    # This might break values with spaces INSIDE quotes if not handled carefully,
    # but for minimal init without python/arrays, this is the trade-off.
    # To support spaces in keys, we recommend URL-enc or underscores.
    
    # Robust shell Loop for parsing space-separated args
    if [ -r "$CMDLINE_FILE" ]; then
        for token in $(cat "$CMDLINE_FILE"); do
            case "$token" in
                ${key}=*)
                    val="${token#*=}"
                    # Strip surrounding quotes (" or ')
                    val=$(echo "$val" | sed -e 's/^"//' -e 's/"$//' -e "s/^'//" -e "s/'$//")
                    echo "$val"
                    return 0
                    ;;
            esac
        done
    fi
    return 1
}

# -----------------------------------------------------------------------------
# Operations
# -----------------------------------------------------------------------------

configure_hostname() {
    local current_hn
    local target_hn="$1"
    
    # Check current hostname
    if has_cmd hostname; then
        current_hn=$(hostname)
    else
        current_hn=$(cat /etc/hostname 2>/dev/null || true)
    fi

    if [ "$current_hn" = "$target_hn" ]; then
        log "Hostname already set to '$target_hn'."
        return
    fi
    
    log "Setting hostname to '$target_hn'..."
    if has_cmd hostname; then
        hostname "$target_hn"
    fi
    
    # Persist
    echo "$target_hn" > /etc/hostname
    
    # Update /etc/hosts (Idempotent)
    if [ -f /etc/hosts ]; then
        # Remove existing entry for 127.0.0.1 <hostname> if different (simplified)
        # Actually safer to jsut append/ensure execution
        if ! grep -q "127.0.0.1.*$target_hn" /etc/hosts; then
             echo "127.0.0.1 $target_hn" >> /etc/hosts
        fi
    fi
}

inject_ssh_key() {
    local raw_key="$1"
    local user="$2"
    local home_dir
    local pwent
    
    # Handle space substitution (underscores to spaces)
    # This is a workaround for kernel command line space limitations
    local key_content=$(echo "$raw_key" | sed 's/_/ /g')
    
    # Validate User
    if ! id "$user" >/dev/null 2>&1; then
        log "User '$user' not found. Creating..."
        if has_cmd adduser; then
             adduser -D "$user" || error "Failed to create user $user"
        elif has_cmd useradd; then
             useradd -m "$user" || error "Failed to create user $user"
        else
             error "Cannot create user $user: no adduser/useradd found."
        fi
    fi

    # Determine Home Directory safely
    # getent is ideal, but fallback to eval ~user or /etc/passwd grep
    if has_cmd getent; then
        home_dir=$(getent passwd "$user" | cut -d: -f6)
    else
        # Fallback for minimal systems
        home_dir=$(grep "^$user:" /etc/passwd | cut -d: -f6)
    fi

    if [ -z "$home_dir" ]; then 
        error "Could not determine home directory for $user"
    fi

    local ssh_dir="$home_dir/.ssh"
    local auth_file="$ssh_dir/authorized_keys"

    # Create directory (idempotent)
    if [ ! -d "$ssh_dir" ]; then
        mkdir -p "$ssh_dir"
        chmod 700 "$ssh_dir"
        chown "$user:$user" "$ssh_dir"
    fi

    # Inject Key (Idempotent)
    if [ -f "$auth_file" ] && grep -Fq "$key_content" "$auth_file"; then
        log "SSH key already authorized for $user."
    else
        log "Authorizing SSH key for $user..."
        echo "$key_content" >> "$auth_file"
        chmod 600 "$auth_file"
        chown "$user:$user" "$auth_file"
    fi
}

# -----------------------------------------------------------------------------
# Main Execution
# -----------------------------------------------------------------------------

main() {
    # 0. Check Environment
    if [ ! -r "$CMDLINE_FILE" ]; then
        echo "Warning: $CMDLINE_FILE not readable. Skipping."
        exit 0
    fi

    # 1. Atomic Lock check
    if ! mkdir "$LOCK_DIR" 2>/dev/null; then
        log "Lock directory $LOCK_DIR exists. Assuming already run. Exiting."
        exit 0
    fi
    # Ensure lock cleanup on exit (optional, but for init script usually we want it to stay locked until reboot)
    # trap 'rm -rf "$LOCK_DIR"' EXIT

    log "Starting..."

    # 2. Hostname
    local param_hostname
    if param_hostname=$(get_param "hostname"); then
        configure_hostname "$param_hostname"
    fi

    # 3. SSH Key
    local param_ssh_key
    if param_ssh_key=$(get_param "ssh_key"); then
        local target_user
        target_user=$(get_param "user" || echo "root")
        inject_ssh_key "$param_ssh_key" "$target_user"
    fi

    log "Completed successfully."
}

main
