# 🪺 nido

**Where your local VMs come to life.**

> *"Why did the VM cross the road? To get to the other hypervisor. But with nido, it just spawns there instantly."* 🐣

---

`nido` is a minimal, automation-friendly toolkit to manage headless KVM virtual machines. It is designed for developers and AI agents who need a fast, simple, and robust way to spawn and control local VMs for testing, development, and sandboxing.

It combines a simple CLI for human operators with a powerful **Model Context Protocol (MCP) Server**, making it the first local-first, AI-native VM manager.

## Philosophy

`nido` is built around **compressed template backups**. Templates are stored as highly compressed qcow2 files, and new VMs are created by rapidly expanding a template into a fresh disk. This keeps storage usage minimal while allowing for near-instant VM deployment.

> *Think of it as a bird's nest (nido 🇮🇹) where your VMs hatch quickly and fly away when done.*

## Features

| Feature | Status | Description |
|---|:---:|---|
| CLI for VM lifecycle | ✅ | Create, start, stop, destroy VMs |
| Compressed templates | ✅ | Fast deployment from minimal storage |
| MCP Server | 🚧 | AI agent integration via Model Context Protocol |
| REST API | 📋 | Traditional HTTP API for integrations |
| Webhooks | 📋 | Event-driven notifications |
| Snapshot management | 📋 | Save and restore VM states |
| GitHub Action | 📋 | CI/CD integration |

**Legend:** ✅ Done | 🚧 In Progress | 📋 Planned

---

## Roadmap

### Phase 1: Foundation ✅
*The egg has hatched!*

- [x] Core CLI (`nido spawn`, `start`, `stop`, `destroy`)
- [x] Compressed template system (`.compact.qcow2`)
- [x] Configuration management (`config.env`)
- [x] VM info and listing
- [x] SSH command generation
- [x] Preflight checks and validation
- [x] Selftest mode (mock & real)

### Phase 2: AI Integration 🚧
*Teaching the bird to talk with robots*

- [ ] MCP Server implementation
  - [ ] Transport: stdio (for Claude Desktop)
  - [ ] Transport: HTTP + SSE (for remote clients)
- [ ] MCP Tools
  - [ ] `vm_list` - List all VMs
  - [ ] `vm_create` - Create VM from template
  - [ ] `vm_start` / `vm_stop` - Lifecycle control
  - [ ] `vm_destroy` - Remove VM
  - [ ] `vm_exec` - Execute commands in VM
  - [ ] `vm_info` - Get VM details
  - [ ] `template_list` - List available templates
- [ ] MCP Resources (dynamic context for AI)
- [ ] MCP Prompts (workflow templates)

### Phase 3: Integration & Ecosystem 📋
*Building the whole tree*

- [ ] REST API Server
  - [ ] CRUD endpoints for VMs
  - [ ] Template management endpoints
  - [ ] Authentication (API keys)
- [ ] Webhook System
  - [ ] Event: `vm.created`, `vm.started`, `vm.stopped`
  - [ ] Event: `vm.destroyed`, `vm.crashed`
  - [ ] Webhook management CLI
- [ ] CI/CD Integration
  - [ ] Official GitHub Action
  - [ ] GitLab CI template
- [ ] File operations
  - [ ] `vm_upload` - Upload files to VM
  - [ ] `vm_download` - Download files from VM

### Phase 4: Advanced Features 📋
*The bird learns new tricks*

- [ ] Snapshot management
  - [ ] Create snapshots
  - [ ] Restore from snapshot
  - [ ] Snapshot listing
- [ ] Advanced networking
  - [ ] Port forwarding
  - [ ] Custom network configurations
- [ ] Template marketplace/registry
- [ ] Fleet management (manage multiple VMs)
- [ ] Interactive TUI (Text User Interface)

### Phase 5: Polish & DX 📋
*Making it shine*

- [ ] Self-healing and auto-recovery
- [ ] AI-powered troubleshooting
- [ ] Comprehensive documentation
- [ ] Plugin system
- [ ] Windows/macOS support (via remote libvirt)

---

## Quick Start

### 1. Installation (The easy way 🐣)

Run the one-line installer on Linux or macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/your-org/nido/main/get-nido.sh | bash
```

### 2. Manual Installation

If you prefer to install manually:

```bash
# 1. Clone the repository
git clone https://github.com/your-org/nido.git
cd nido

# 2. Install prerequisites (one-time)
# Supports: Debian, Ubuntu, Fedora, and macOS (via Homebrew)
./bin/setup_deps.sh

# 3. Configure
cp config/config.example.env config/config.env
# Edit config/config.env with your local paths
```

### 3. Usage

```bash
#3) Create a compressed template from a base VM (VM must be shut off):
   - `./bin/nido template debian-iso-1 template-headless`
4) Create a VM:
   - `./bin/nido create vm-test-1`
5) Start and wait for IP:
   - `./bin/nido start vm-test-1`
6) OR spawn (create + start) in one go:
   - `./bin/nido spawn vm-test-1`
# -> SSH command will be printed, ready to copy-paste
```


## Commands

```bash
spawn <name> [template]      # Create and start a new VM
create <name> [template]     # Just create VM disk and define libvirt domain
start <name>                 # Start an existing VM
stop <name>                  # Shutdown a running VM
delete <name>                # Remove VM and delete its disk volume 💀
ls [regex]                   # List VMs (matches all by default)
info <name>                  # Print VM IP and SSH connection string
prune                        # Delete orphan volumes in pool vms
template <src> <tpl>         # Create a compressed backup from a source VM
selftest                     # Run automated tests
mcp-server start             # Start MCP server (coming soon™)
```

## Why nido?

| | nido | Vagrant | Multipass | E2B |
|---|:---:|:---:|:---:|:---:|
| **AI Integration** | 🚧 MCP | ❌ | ❌ | ✅ Cloud |
| **Local-first** | ✅ | ✅ | ✅ | ❌ |
| **Storage** | Compressed | Full boxes | Full images | Cloud |
| **Simplicity** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Cost** | Free | Free | Free | $$$  |
| **Fun to use** | 🐣 | 🤷 | 🤷 | 🤷 |

## Configuration

Default config file: `config/config.env`

```bash
POOL_PATH=/path/to/libvirt-pool
BACKUP_DIR=/path/to/backups
VMS_POOL=vms
TEMPLATE_DEFAULT=template-headless
VM_MEM_MB=2048
VM_VCPUS=2
VM_OS_VARIANT=debian12
NETWORK_HOSTONLY=hostonly56
NETWORK_NAT=default
SSH_USER=vmuser
WAIT_TIMEOUT=60
```

## Contributing

Contributions are welcome! Whether it's a bug fix, new feature, or just improving docs, feel free to open a PR.

> *Every bird in the nest helps it grow.* 🪺

## License

MIT License - do whatever you want, just don't blame us if your VMs fly away.

---

<p align="center">
  <i>Made with ❤️ and a healthy dose of <code>virsh</code> commands</i>
</p>
