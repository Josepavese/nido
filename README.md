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

## 💡 Tips & Tricks (The "Matryoshka" Section)

### 🪺 VM inside VM (Nested Virtualization)

Yes, you can hatch a bird inside a bird. This is perfect for testing hypervisors or creating complex labs.

- **Prerequisite:** Set `VM_NESTED=true` in your `config.env`.
- **How it works:** We pass `--cpu host-passthrough` to the guest, giving it the raw power (and VMX/SVM flags) of your physical CPU.
- **Warning:** Expect a small performance hit. It's like inception, the deeper you go, the slower time (and your VM) becomes.

### 🐳 VM inside Container (KVM-in-Docker)

Want to lanciare una VM da dentro un container? `nido` non giudica i tuoi feticci tecnologici.

- **Prerequisite:** Il tuo container deve essere lanciato con `--privileged` e avere accesso a `/dev/kvm` (es. `--device /dev/kvm`).
- **Setup:** Installa `libvirt-daemon-system` e `nido` nel container. Avvia `virtlogd` e `libvirtd` all'entrypoint.
- **Pro Tip:** È il modo più pulito per avere un ambiente di automazione VM portatile senza sporcare il tuo host.

---

## Quick Start

### 1. Installation (Recommended)

#### a) Installer script (Linux/macOS)

Scarica l'installer dalla cartella `bin/installers` del repository oppure usa il comando:

```bash
git clone https://github.com/Josepavese/nido.git
cd nido
# Esegui l'installer locale:
./bin/installers/get-nido.sh
```

> **Nota:** Gli installer ufficiali sono sempre disponibili nella cartella `bin/installers` del repository. Puoi copiarli e distribuirli su altri sistemi.

#### b) Installazione manuale

Se preferisci installare manualmente:

```bash
git clone https://github.com/your-org/nido.git
cd nido
# Installa le dipendenze (Debian/Ubuntu/Fedora/macOS):
./bin/setup_deps.sh
# Configura l'ambiente:
cp config/config.example.env config/config.env
# Modifica config/config.env con i tuoi percorsi locali
```

### 2. Usage

```bash
# 1) Crea un template compresso da una VM base (la VM deve essere spenta):
  ./bin/nido template debian-iso-1 template-headless
# 2) Crea una nuova VM:
  ./bin/nido create vm-test-1
# 3) Avvia la VM e attendi l'IP:
  ./bin/nido start vm-test-1
# 4) Oppure "spawn" (crea + avvia in un colpo solo):
  ./bin/nido spawn vm-test-1
# -> Il comando SSH verrà stampato, pronto da copiare
```

## Installer distribution

Gli installer ufficiali (es. `get-nido.sh`) sono mantenuti nella cartella [`bin/installers`](bin/installers/). Per distribuire nido su altri sistemi, copia semplicemente lo script desiderato da questa cartella e segui le istruzioni di installazione.

Se aggiorni l'installer, ricordati di committare la nuova versione in git e aggiornare eventuali riferimenti nei README.

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
