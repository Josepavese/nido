# VM Ops - Architettura Dual-Mode (CLI + MCP)

## Vision

VM Ops diventa il **tool di riferimento per AI agents che devono gestire VM locali**. Mantiene la semplicità CLI per uso umano, aggiungendo un layer MCP per integrazione seamless con AI assistants.

```
┌─────────────────────────────────────────────────────────────────┐
│                         VM OPS                                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────────┐                    ┌─────────────────────┐   │
│   │   CLI Mode  │                    │     MCP Server      │   │
│   │  (Human)    │                    │   (AI Agents)       │   │
│   └──────┬──────┘                    └──────────┬──────────┘   │
│          │                                      │               │
│          │         ┌──────────────┐             │               │
│          └────────▶│   Core API   │◀────────────┘               │
│                    │  (Shared)    │                             │
│                    └──────┬───────┘                             │
│                           │                                     │
│                    ┌──────▼───────┐                             │
│                    │   libvirt    │                             │
│                    │    /KVM      │                             │
│                    └──────────────┘                             │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## Architettura a Livelli

### Layer 1: Interface Layer

#### 1.1 CLI Interface (Esistente)
```bash
nido create vm-test-1
nido spawn vm-test-1
nido ls
```

#### 1.2 MCP Server Interface (Nuovo)
```bash
nido mcp-server start          # Avvia MCP server (stdio)
nido mcp-server start --http   # Avvia MCP server (HTTP + SSE)
```

### Layer 2: Core API (Shared Logic)

Logica condivisa tra CLI e MCP, implementata come moduli riusabili:

```
core/
├── vm_manager.sh       # Gestione lifecycle VM
├── template_manager.sh # Gestione template compressi
├── network_manager.sh  # Configurazione rete
├── storage_manager.sh  # Gestione dischi
└── monitor.sh          # Monitoring e health checks
```

### Layer 3: Backend (libvirt/KVM)

Interfaccia diretta con libvirt tramite `virsh` e API.

---

## MCP Server - Design Dettagliato

### Transport Supportati

| Transport | Use Case | Comando |
|-----------|----------|---------|
| **stdio** | Claude Desktop, locale | `nido mcp-server` |
| **HTTP+SSE** | Integrazione remota | `nido mcp-server --http --port 8080` |

### Tools Esposti

#### Categoria: VM Lifecycle

```json
{
  "name": "vm_list",
  "title": "List Virtual Machines",
  "description": "Returns a list of all VMs with their current status",
  "inputSchema": {
    "type": "object",
    "properties": {
      "filter": {
        "type": "string",
        "description": "Optional name pattern to filter VMs (e.g., 'test-*')"
      }
    }
  },
  "outputSchema": {
    "type": "object",
    "properties": {
      "vms": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name": { "type": "string" },
            "status": { "type": "string" },
            "ip": { "type": "string" },
            "cpu": { "type": "number" },
            "memory_mb": { "type": "number" }
          }
        }
      },
      "count": { "type": "number" }
    }
  }
}
```

```json
{
  "name": "vm_create",
  "title": "Create New VM",
  "description": "Creates a new VM from a compressed template. Returns connection info when ready.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": {
        "type": "string",
        "description": "Unique name for the VM (e.g., 'vm-test-1')"
      },
      "template": {
        "type": "string",
        "description": "Template to use (default: template-headless)"
      },
      "profile": {
        "type": "string",
        "enum": ["default", "minimal", "development", "ml-training"],
        "description": "Resource profile"
      },
      "ephemeral": {
        "type": "boolean",
        "description": "If true, VM auto-destroys after timeout"
      },
      "timeout_hours": {
        "type": "number",
        "description": "Hours before auto-destroy (if ephemeral)"
      }
    },
    "required": ["name"]
  },
  "outputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" },
      "status": { "type": "string" },
      "ip": { "type": "string" },
      "ssh_command": { "type": "string" },
      "ssh_user": { "type": "string" },
      "creation_time_seconds": { "type": "number" }
    }
  },
  "annotations": {
    "destructive": false,
    "idempotent": false,
    "long_running": true
  }
}
```

```json
{
  "name": "vm_start",
  "title": "Start VM",
  "description": "Starts a stopped VM and returns connection info",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string", "description": "VM name to start" }
    },
    "required": ["name"]
  }
}
```

```json
{
  "name": "vm_stop",
  "title": "Stop VM",
  "description": "Gracefully stops a running VM",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" },
      "force": { "type": "boolean", "description": "Force stop if graceful fails" }
    },
    "required": ["name"]
  }
}
```

```json
{
  "name": "vm_delete",
  "title": "Delete VM",
  "description": "Permanently deletes a VM and its disk. This action is irreversible.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" }
    },
    "required": ["name"]
  },
  "annotations": {
    "destructive": true,
    "confirmation_required": true
  }
}
```

```json
{
  "name": "vm_info",
  "title": "Get VM Info",
  "description": "Returns detailed information about a specific VM",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" }
    },
    "required": ["name"]
  },
  "outputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" },
      "status": { "type": "string" },
      "ip": { "type": "string" },
      "hostname": { "type": "string" },
      "ssh_command": { "type": "string" },
      "cpu_count": { "type": "number" },
      "memory_mb": { "type": "number" },
      "disk_gb": { "type": "number" },
      "template": { "type": "string" },
      "created_at": { "type": "string" },
      "uptime_seconds": { "type": "number" }
    }
  }
}
```

#### Categoria: Command Execution

```json
{
  "name": "vm_exec",
  "title": "Execute Command in VM",
  "description": "Executes a shell command inside the VM via SSH and returns output",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string", "description": "VM name" },
      "command": { "type": "string", "description": "Shell command to execute" },
      "timeout_seconds": { "type": "number", "description": "Command timeout (default: 60)" },
      "working_dir": { "type": "string", "description": "Working directory for command" }
    },
    "required": ["name", "command"]
  },
  "outputSchema": {
    "type": "object",
    "properties": {
      "stdout": { "type": "string" },
      "stderr": { "type": "string" },
      "exit_code": { "type": "number" },
      "execution_time_ms": { "type": "number" }
    }
  },
  "annotations": {
    "potentially_dangerous": true
  }
}
```

```json
{
  "name": "vm_upload",
  "title": "Upload File to VM",
  "description": "Uploads a file from host to VM",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" },
      "local_path": { "type": "string" },
      "remote_path": { "type": "string" }
    },
    "required": ["name", "local_path", "remote_path"]
  }
}
```

```json
{
  "name": "vm_download",
  "title": "Download File from VM",
  "description": "Downloads a file from VM to host",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" },
      "remote_path": { "type": "string" },
      "local_path": { "type": "string" }
    },
    "required": ["name", "remote_path", "local_path"]
  }
}
```

#### Categoria: Template Management

```json
{
  "name": "template_list",
  "title": "List Templates",
  "description": "Returns available VM templates",
  "inputSchema": { "type": "object", "properties": {} },
  "outputSchema": {
    "type": "object",
    "properties": {
      "templates": {
        "type": "array",
        "items": {
          "type": "object",
          "properties": {
            "name": { "type": "string" },
            "size_mb": { "type": "number" },
            "os": { "type": "string" },
            "description": { "type": "string" }
          }
        }
      }
    }
  }
}
```

```json
{
  "name": "template_create",
  "title": "Create Template from VM",
  "description": "Creates a compressed template from an existing VM",
  "inputSchema": {
    "type": "object",
    "properties": {
      "source_vm": { "type": "string", "description": "Source VM name" },
      "template_name": { "type": "string", "description": "Name for new template" },
      "description": { "type": "string" }
    },
    "required": ["source_vm", "template_name"]
  },
  "annotations": {
    "long_running": true
  }
}
```

#### Categoria: Snapshot & Recovery

```json
{
  "name": "vm_snapshot",
  "title": "Create VM Snapshot",
  "description": "Creates a snapshot of VM state for recovery",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" },
      "snapshot_name": { "type": "string" },
      "description": { "type": "string" }
    },
    "required": ["name", "snapshot_name"]
  }
}
```

```json
{
  "name": "vm_restore",
  "title": "Restore VM from Snapshot",
  "description": "Restores VM to a previous snapshot state",
  "inputSchema": {
    "type": "object",
    "properties": {
      "name": { "type": "string" },
      "snapshot_name": { "type": "string" }
    },
    "required": ["name", "snapshot_name"]
  },
  "annotations": {
    "destructive": true
  }
}
```

---

## Resources Esposti (MCP)

Oltre ai tools, VM Ops espone risorse per contesto:

```json
{
  "uri": "vmops://vms",
  "name": "VM List",
  "description": "Current list of all VMs and their status",
  "mimeType": "application/json"
}
```

```json
{
  "uri": "vmops://vm/{name}/logs",
  "name": "VM Logs",
  "description": "Recent logs from the specified VM",
  "mimeType": "text/plain"
}
```

```json
{
  "uri": "vmops://templates",
  "name": "Template Catalog",
  "description": "Available VM templates",
  "mimeType": "application/json"
}
```

---

## Prompts Predefiniti (MCP)

```json
{
  "name": "deploy_and_test",
  "title": "Deploy and Test Application",
  "description": "Creates a VM, deploys code, runs tests, and reports results",
  "arguments": [
    { "name": "app_name", "description": "Application name", "required": true },
    { "name": "git_repo", "description": "Git repository URL", "required": true },
    { "name": "test_command", "description": "Command to run tests", "required": true }
  ]
}
```

```json
{
  "name": "quick_sandbox",
  "title": "Quick Sandbox",
  "description": "Creates an ephemeral VM for quick testing",
  "arguments": [
    { "name": "duration", "description": "How long to keep the VM (e.g., '1h', '30m')", "required": false }
  ]
}
```

---

## Configurazione MCP Client

### Claude Desktop (config.json)

```json
{
  "mcpServers": {
    "vmops": {
      "command": "/path/to/nido",
      "args": ["mcp-server"],
      "env": {
        "VMOPS_TEMPLATES_DIR": "/var/lib/nido/templates",
        "VMOPS_VMS_DIR": "/var/lib/nido/vms"
      }
    }
  }
}
```

### Cursor/VSCode

```json
{
  "mcp.servers": {
    "vmops": {
      "command": "nido",
      "args": ["mcp-server"]
    }
  }
}
```

---

## Progress Notifications

Per operazioni lunghe (creazione VM, template), il server invia progress:

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/progress",
  "params": {
    "progressToken": "create-vm-123",
    "progress": 45,
    "total": 100,
    "message": "Decompressing template..."
  }
}
```

---

## Error Handling

Errori strutturati per debugging:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32001,
    "message": "VM not found",
    "data": {
      "vm_name": "vm-test-1",
      "suggestion": "Use vm_list to see available VMs"
    }
  }
}
```

### Error Codes

| Code | Name | Description |
|------|------|-------------|
| -32001 | VM_NOT_FOUND | VM does not exist |
| -32002 | VM_ALREADY_EXISTS | VM name already in use |
| -32003 | TEMPLATE_NOT_FOUND | Template does not exist |
| -32004 | VM_NOT_RUNNING | VM must be running for this operation |
| -32005 | INSUFFICIENT_RESOURCES | Not enough CPU/RAM/disk |
| -32006 | SSH_CONNECTION_FAILED | Cannot connect to VM via SSH |
| -32007 | COMMAND_TIMEOUT | Command execution timed out |
| -32008 | PERMISSION_DENIED | Operation not allowed |

---

## Security Considerations

### 1. Tool Annotations
Ogni tool ha annotations per indicare:
- `destructive`: Operazione irreversibile
- `confirmation_required`: Richiede conferma utente
- `potentially_dangerous`: Può eseguire codice arbitrario

### 2. Rate Limiting
```bash
nido mcp-server --rate-limit 100/min
```

### 3. Allowed Commands Whitelist
Per `vm_exec`, opzionale whitelist di comandi permessi:
```bash
nido mcp-server --allowed-commands "apt,pip,npm,git"
```

### 4. Audit Logging
Tutte le operazioni MCP vengono loggate:
```
/var/log/vmops/mcp-audit.log
```

---

## Implementazione Suggerita

### Fase 1: MCP Server Base (2-3 settimane)
- [ ] Implementare JSON-RPC 2.0 handler
- [ ] Transport stdio
- [ ] Tools: vm_ls, vm_create, vm_spawn, vm_stop, vm_delete, vm_info
- [ ] Error handling base

### Fase 2: Command Execution (1-2 settimane)
- [ ] vm_exec con timeout
- [ ] vm_upload, vm_download
- [ ] Progress notifications

### Fase 3: Template & Snapshot (1-2 settimane)
- [ ] template_list, template_create
- [ ] vm_snapshot, vm_restore
- [ ] Resources MCP

### Fase 4: HTTP Transport & Polish (1-2 settimane)
- [ ] HTTP + SSE transport
- [ ] Prompts predefiniti
- [ ] Audit logging
- [ ] Documentation

---

## File Structure Proposta

```
vmops/
├── nido                    # Entry point CLI
├── mcp/
│   ├── server.sh               # MCP server main
│   ├── handlers/
│   │   ├── tools.sh            # Tool handlers
│   │   ├── resources.sh        # Resource handlers
│   │   └── prompts.sh          # Prompt handlers
│   ├── transport/
│   │   ├── stdio.sh            # stdio transport
│   │   └── http.sh             # HTTP + SSE transport
│   └── schema/
│       └── tools.json          # Tool definitions
├── core/
│   ├── vm_manager.sh
│   ├── template_manager.sh
│   └── ...
├── config/
│   └── vmops.conf
└── docs/
    ├── mcp-integration.md
    └── api-reference.md
```

