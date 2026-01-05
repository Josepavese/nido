# 🪺 nido

**Where your local VMs come to life.**

> *"Why did the VM cross the road? To get to the other hypervisor. But with nido, it just spawns there instantly."* 🐣

---

`nido` is a minimal, automation-friendly toolkit to manage local KVM virtual machines (Headless or GUI). It is designed for developers and AI agents who need a fast, simple, and robust way to spawn and control local VMs for testing, development, and sandboxing.

It combines a simple CLI for human operators with a powerful **Model Context Protocol (MCP) Server**, making it the first local-first, AI-native VM manager.

## Philosophy

`nido` is built around **compressed template backups**. Templates are stored as highly compressed qcow2 files, and new VMs are created by rapidly expanding a template into a fresh disk. This keeps storage usage minimal while allowing for near-instant VM deployment.

> *Think of it as a bird's nest ("nido") where your VMs hatch quickly and fly away when done.*

## Features

| Feature | Status | Description |
|---|:---:|---|
| CLI for VM lifecycle | ✅ | Create, start, stop, destroy VMs |
| Compressed templates | ✅ | Fast deployment from minimal storage |
| MCP Server | ✅ | AI agent integration via Model Context Protocol |
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
- [x] Interactive Setup Wizard (`nido setup`)
- [x] VM info and listing
- [x] SSH command generation
- [x] Preflight checks and validation
- [x] Selftest mode (mock & real)
- [x] IDE Registration helper (`nido register`)

### Phase 2: AI Integration 🚧

*Teaching the bird to talk with robots*

- [x] MCP Server implementation
  - [x] Transport: stdio (for Claude Desktop)
  - [ ] Transport: HTTP + SSE (for remote clients)
- [x] MCP Tools
  - [x] `vm_list` - List all VMs
  - [x] `vm_create` - Create VM from template
  - [x] `vm_start` / `vm_stop` - Lifecycle control
  - [x] `vm_destroy` - Remove VM (implemented as `vm_delete`)
  - [ ] `vm_exec` - Execute commands in VM
  - [x] `vm_info` - Get VM details
  - [ ] `template_list` - List available templates
  - [x] `config_get` / `config_set` - Manage configuration
  - [x] `nido_describe` - System overview
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

## 💡 Tips & Tricks (The "Matryoshka" Section)

### 🪺 VM inside VM (Nested Virtualization)

Yes, you can hatch a bird inside a bird. This is perfect for testing hypervisors or creating complex labs.

- **Prerequisite:** Set `VM_NESTED=true` in your `config.env`.
- **How it works:** We pass `--cpu host-passthrough` to the guest, giving it the raw power (and VMX/SVM flags) of your physical CPU.
- **Warning:** Expect a small performance hit. It's like inception, the deeper you go, the slower time (and your VM) becomes.

### 🐳 VM inside Container (KVM-in-Docker)

Want to hatch a bird inside a whale? Nido doesn't judge your technical curiosities.

- **Prerequisite:** Your container must be launched with `--privileged` and have access to `/dev/kvm` (e.g., `--device /dev/kvm`).
- **Setup:** Install `libvirt-daemon-system` and `nido` inside the container. Start `virtlogd` and `libvirtd` at the entrypoint.
- **Pro Tip:** This is the cleanest way to have a portable VM automation environment without cluttering your host system.

---

## Quick Start

### 1. Installation (Recommended)

#### a) Installer script (Linux/macOS)

Download the installer from the `bin/installers` directory of the repository or use the command:

```bash
git clone https://github.com/Josepavese/nido.git
cd nido
# Run the local installer:
./bin/installers/get-nido.sh
```

> **Note:** Official installers are always available in the `bin/installers` folder of the repository. You can copy and distribute them to other systems.

#### b) Manual Installation

If you prefer to install manually:

```bash
git clone https://github.com/your-org/nido.git
cd nido
# Install dependencies (Debian/Ubuntu/Fedora/macOS):
./bin/setup_deps.sh
# Configure the environment:
cp config/config.example.env config/config.env
# Edit config/config.env with your local paths
```

### 2. Usage

```bash
# 1) Create a compressed template from a base VM (VM must be shut off):
  nido template debian-iso-1 template-headless
# 2) Create a new VM:
  nido create vm-test-1
# 3) Start the VM and wait for the IP:
  nido start vm-test-1
# 4) Or "spawn" (create + start in one go):
  nido spawn vm-test-1
# -> The SSH command will be printed, ready to be copied
```

Official installers (e.g., `get-nido.sh`) are maintained in the [`bin/installers`](bin/installers/) folder. To distribute nido to other systems, simply copy the desired script from this folder and follow the installation instructions.

If you update the installer, remember to commit the new version to git and update any references in the READMEs.

## Commands

```bash
nido spawn <name> [template]      # Create and start a new VM
nido create <name> [template]     # Just create VM disk and define libvirt domain
nido start <name>                 # Start an existing VM
nido stop <name>                  # Shutdown a running VM
nido delete <name>                # Remove VM and delete its disk volume 💀
nido ls [regex]                   # List VMs (matches all by default)
nido info <name>                  # Print VM IP and SSH connection string
nido prune                        # Delete orphan volumes in pool vms
nido template <src> <tpl>         # Create a compressed backup from a source VM
nido setup                        # Interactive configuration wizard
nido config                       # View current configuration
nido register                     # Generate MCP config for IDEs
nido selftest                     # Run automated tests
nido version                      # Show version info
nido mcp-server start             # Start MCP server
```

## Why nido?

| | nido | Vagrant | Multipass | E2B |
|---|:---:|:---:|:---:|:---:|
| **AI Integration** | ✅ MCP | ❌ | ❌ | ✅ Cloud |
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
