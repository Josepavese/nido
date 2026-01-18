# Nido MCP Server

The **Nido MCP Server** enables AI agents (like Claude) to inspect, control, and manage your local specific virtual machine environment directly. It implements the Model Context Protocol (MCP) over standard input/output (stdio).

## Overview

This server exposes a rich set of tools to:

- **Lifecycle**: Create, start, stop, and delete VMs.
- **Inspection**: Get details on VM state, IPs, and SSH connection strings.
- **Templates**: Create and manage cold storage templates of your VMs.
- **Images**: Search, pull, and manage the local cache of cloud images (e.g., Ubuntu, Debian).
- **Maintenance**: Prune stopped VMs, check system health (`doctor`), and clean up caches.
 - **Help**: `mcp-help` emits the full MCP tools catalog (names, descriptions, schemas).

## Installation & Configuration

To use this with an MCP client (e.g., Claude Desktop, Zed), add the following to your `mcp_config.json` (or equivalent):

```json
{
  "mcpServers": {
    "nido": {
      "command": "nido",
      "args": ["mcp"]
    }
  }
}
```

Ensure the `nido` binary is in your system PATH.

## Available Tools

### VM Lifecycle

#### `vm_list`

List all virtual machines currently known to the nest.

- **Returns**: Array of VM objects (Name, State, PID, Ports).

#### `vm_create`

Hatch a new virtual machine.

- **Args**:
  - `name` (string, required): Unique identifier for the VM.
  - `image` (string): Cloud image tag (e.g., `ubuntu:24.04`).
  - `template` (string): Name of a template to clone from.
  - `gui` (boolean): Enable VNC visual output.
  - `user_data` (string): Cloud-init user-data content.
  - `ports` (array, optional): *[Planned]* List of port forwarding rules (e.g. `["8080:80", "http:80"]`).

#### `vm_start`

Wake up a sleeping VM.

- **Args**:
  - `name` (string, required)
  - `gui` (boolean)

#### `vm_stop`

Send ACPI shutdown signal to the VM.

- **Args**:
  - `name` (string, required)

#### `vm_delete`

Permanently remove a VM and its disk.

- **Args**:
  - `name` (string, required)

### Inspection & Access

#### `vm_info`

Get detailed JSON status of a specific VM.

- **Args**: `name`
- **Returns**: IP address, SSH port, VNC port, State, etc.

#### `vm_ssh`

Get the connection string to SSH into the VM.

- **Args**: `name`
- **Returns**: String (e.g., `ssh -p 32001 user@127.0.0.1`)

#### `vm_doctor`

Run diagnostics on the Nido environment (QEMU path, permissions, directories).

### Template Management

#### `vm_template_list`

List available templates in cold storage.

#### `vm_template_create`

Archive a VM into a compressed template.

- **Args**: `vm_name`, `template_name`

#### `vm_template_delete`

Delete a template.

- **Args**: `name`

### Image & Cache Management

#### `vm_images_list`

View the catalog of available cloud images (Official & Flavours).

#### `vm_images_pull`

Download an image into the local cache.

- **Args**: `image` (e.g., `alma:9`)

#### `vm_images_update`

Force refresh the upstream image catalog.

#### `vm_images_info`

Inspect catalog metadata for a specific image tag (name:version or alias).

#### `vm_images_remove`

Remove a cached image by `name:version`.

#### `vm_cache_list` / `vm_cache_info`

Inspect the local image cache usage.

#### `vm_cache_remove`

Delete a specific image from cache.

- **Args**: `image` (name:version)

#### `vm_cache_prune`

Clean up unused or all cached images.

- **Args**: `unused_only` (bool)

---

## Planned Features (Port Forwarding)

The following tools are designed for the upcoming Advanced Port Forwarding system:

#### `vm_port_forward`

Add a new port mapping to a running or stopped VM.

- **Args**:
  - `name` (string): VM Name.
  - `guest_port` (int): Port inside the VM.
  - `host_port` (int, optional): Port on host (0 for auto).
  - `label` (string, optional): Human readable label.
  - `protocol` (string): "tcp" or "udp" (default "tcp").

#### `vm_port_unforward`

Remove a port mapping.

- **Args**:
  - `name` (string)
  - `identifier` (string): Label or Guest Port number.

#### `vm_port_list`

List active port forwardings for a VM.

- **Args**: `name`
