# 🪺 nido

**Where your local VMs come to life.**

_"Why did the VM cross the road? To get to the other hypervisor. But with nido, it just spawns there instantly."_ 🐣

---

`nido` is a minimal, automation-friendly toolkit to manage local virtual machines on **Linux, macOS, and Windows**. It's designed for developers and AI agents who need a fast, simple, and robust way to spawn and control local VMs for testing, development, and sandboxing.

It combines a simple CLI for human operators with a powerful **Model Context Protocol (MCP) Server**, making it the first local-first, AI-native VM manager.

## Philosophy

nido is built around **compressed template backups**. Templates are stored as highly compressed `.compact.qcow2` files, and new VMs are created by rapidly expanding a template into a fresh disk. This keeps storage usage minimal while allowing for near-instant VM deployment.

Think of it as a bird's nest ("nido") where your VMs hatch quickly and fly away when done. 🐦

## Features

Legend: ✅ Done | 🚧 In Progress | 📋 Planned

### Core VM Management ✅

- **Instant Spawn:** Deploy VMs from compressed templates in seconds
- **Lifecycle Control:** Start, stop, delete VMs with simple commands
- **Template System:** Archive running VMs into reusable templates
- **SSH Integration:** Automatic SSH connection strings
- **Health Checks:** Built-in diagnostics (`nido doctor`)
- **Cross-Platform:** Native QEMU support on Linux (KVM), macOS (HVF), and Windows (WHPX)

### AI Integration ✅

- **MCP Server:** Full Model Context Protocol implementation
- **12 MCP Tools:** Complete VM management via AI agents
  - `vm_list`, `vm_create`, `vm_start`, `vm_stop`, `vm_delete`
  - `vm_info`, `vm_ssh`, `vm_prune`
  - `template_list`, `template_create`
  - `config_get`, `doctor`
- **Claude Desktop Ready:** Works out of the box with `nido register`
- **Antigravity Compatible:** Seamless integration with modern AI coding assistants

### Developer Experience ✅

- **Zero Dependencies:** No `libvirt` or `virsh` required. Pure QEMU.
- **Configuration Management:** Simple `.env` file for all settings
- **Port Forwarding:** Automatic SSH port mapping (no root needed!)
- **Compressed Storage:** Templates use `.compact.qcow2` format

## Quick Start

### 1. Installation

#### Linux

```bash
git clone https://github.com/Josepavese/nido
cd nido
bash bin/install.sh
source ~/.bashrc
```

#### macOS

```bash
brew install qemu
curl -L https://github.com/Josepavese/nido/releases/latest/download/nido-darwin-amd64 -o nido
chmod +x nido
sudo mv nido /usr/local/bin/
```

#### Windows

```powershell
choco install qemu
# Download from https://github.com/Josepavese/nido/releases
# Extract and add to PATH
```

> **Beta Testers Wanted!** macOS and Windows support is fresh. See [TESTING.md](TESTING.md) for details.

### 2. Usage

```bash
# The essentials
nido spawn my-vm                   # Hatch a new VM from default template
nido ls                            # List all life forms in the nest
nido ssh my-vm                     # Instant bridge via SSH
nido stop my-vm                    # Put VM into deep sleep
nido delete my-vm                  # Evict VM permanently

# Template management
nido template list                 # See what's in cold storage
nido template create my-vm golden  # Archive VM as reusable template

# Power user moves
nido start my-vm                   # Revive a stopped VM
nido prune                         # Vaporize all stopped VMs
nido info my-vm                    # Inspect neural links (IP, Port)
nido doctor                        # Run system health check
nido config                        # View current genetics

# AI agent setup
# AI agent setup
nido register                      # Get MCP config for Claude/Antigravity

# Image Management 🆕
nido image list                    # Browse available cloud images
nido image pull ubuntu:24.04       # Download official Ubuntu image
nido spawn my-vm --image ubuntu:24.04  # Spawn directly from image
```

## Commands

| Command | What it does | Example |
|---------|--------------|---------|
| `spawn <name> [--image <img/tpl>]` | Create and start a VM | `nido spawn vm1 --image ubuntu:24.04` |
| `start <name>` | Revive a stopped VM | `nido start test-vm` |
| `stop <name>` | Put VM into deep sleep | `nido stop test-vm` |
| `delete <name>` | Evict VM permanently | `nido delete test-vm` |
| `prune` | Remove all stopped VMs | `nido prune` |
| `ls` | List all VMs | `nido ls` |
| `info <name>` | Get VM details | `nido info test-vm` |
| `ssh <name>` | SSH into VM | `nido ssh test-vm` |
| `image list` | List cloud images | `nido image list` |
| `image pull <image>` | Download image | `nido image pull ubuntu:24.04` |
| `template list` | List templates | `nido template list` |
| `template create <vm> <tpl>` | Archive VM | `nido template create my-vm golden` |
| `doctor` | System diagnostics | `nido doctor` |
| `config` | View configuration | `nido config` |
| `register` | MCP setup helper | `nido register` |

### Automation

The registry (`registry/images.json`) is automatically updated daily by a GitHub Action.
To add a new image source, submit a PR to `registry/sources.yaml`. The `registry-builder` tool will automatically fetch the latest versions and checksums.

### Configuration

By default, images are stored in `~/.nido/images/`. You can change this via an environment variable:

```bash
export NIDO_IMAGE_DIR="/path/to/my/images"
```

## 💡 Tips & Tricks (The "Matryoshka" Section)

### 🪺 VM inside VM (Nested Virtualization)

Yes, you can hatch a bird inside a bird. This is perfect for testing hypervisors or creating complex labs.

- **How it works:** We pass the right CPU flags to enable nested virtualization
- **Warning:** Expect a small performance hit. It's like inception—the deeper you go, the slower time (and your VM) becomes.

### 🐳 VM inside Container (KVM-in-Docker)

Want to hatch a bird inside a whale? Nido doesn't judge your technical curiosities.

- **Prerequisite:** Container needs `--privileged` and `--device /dev/kvm`
- **Pro Tip:** This is the cleanest way to have a portable VM automation environment without cluttering your host system.

## Why Nido?

| Feature | nido | Vagrant | Multipass | E2B |
|---------|:----:|:-------:|:---------:|:---:|
| **AI Integration** | ✅ MCP | ❌ | ❌ | ✅ Cloud |
| **Local-first** | ✅ | ✅ | ✅ | ❌ |
| **Storage** | Compressed | Full boxes | Full images | Cloud |
| **Cross-Platform** | ✅ | ✅ | ✅ | ✅ |
| **Simplicity** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Humor Level** | 🐣🐣🐣 | 😐 | 😐 | 🤖 |

## Roadmap

### Phase 3: Integration & Ecosystem 📋

Building the whole tree

- REST API Server (CRUD endpoints, webhooks)
- CI/CD Integration (GitHub Actions, GitLab CI)
- File operations (`vm_upload`, `vm_download`)

### Phase 4: Advanced Features 📋

The bird learns new tricks

- Snapshot management (create, restore, list)
- Advanced networking (custom configs)
- Template marketplace/registry
- Fleet management (multiple VMs)
- Interactive TUI

### Phase 5: Polish & DX 📋

Making it shine

- Self-healing and auto-recovery
- AI-powered troubleshooting
- Comprehensive documentation
- Plugin system

## Contributing

Found a bug? Have a feature idea? Want to teach the bird new tricks?

1. Open an issue on GitHub
2. Fork, hack, and submit a PR
3. Follow the [tone of voice guidelines](.tonodivoce) (keep it fun!)

## License

MIT License - See [LICENSE](LICENSE) for details.

---

<p align="center">
  <i>Made with ❤️ for the Agentic future.</i><br>
  <i>"It's not a VM, it's a lifestyle."</i> 🪺
</p>
