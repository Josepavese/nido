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
- **19 MCP Tools:** Complete VM management via AI agents (including GUI/VNC and cache management!)
  - **VM Lifecycle**: `vm_list`, `vm_create`, `vm_start`, `vm_stop`, `vm_delete`, `vm_info`, `vm_ssh`, `vm_prune`
  - **Images**: `vm_images_list`, `vm_images_pull`, `vm_images_update`
  - **Cache**: `vm_cache_list`, `vm_cache_info`, `vm_cache_remove`, `vm_cache_prune`
  - **Templates**: `vm_template_list`, `vm_template_create`
  - **System**: `vm_config_get`, `vm_doctor`
- **Claude Desktop Ready:** Works out of the box with `nido register`
- **Antigravity Compatible:** Seamless integration with modern AI coding assistants

### Developer Experience ✅

- **Zero Dependencies:** No `libvirt` or `virsh` required. Pure QEMU.
- **Configuration Management:** Simple `.env` file for all settings
- **Port Forwarding:** Automatic SSH port mapping (no root needed!)
- **Compressed Storage:** Templates use `.compact.qcow2` format

## Quick Start

### 1. Installation

> **⚡ Lightning-Fast Install** - Just ~4MB download. No Git required!

#### Linux & macOS

```bash
curl -fsSL https://raw.githubusercontent.com/Josepavese/nido/main/installers/quick-install.sh | bash
source ~/.bashrc  # or ~/.zshrc
nido version
```

#### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/Josepavese/nido/main/installers/quick-install.ps1 | iex
# Restart terminal, then:
nido version
```

> **📖 More Options:** See [`installers/README.md`](installers/README.md) for alternative installation methods, including a lightweight build-from-source option for tinkerers who want bleeding-edge features without cloning the entire repository.

#### Manual Installation (All Platforms)

Download the latest binary from [GitHub Releases](https://github.com/Josepavese/nido/releases/latest):

```bash
# Linux
curl -L https://github.com/Josepavese/nido/releases/latest/download/nido-linux-amd64 -o nido
chmod +x nido && sudo mv nido /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/Josepavese/nido/releases/latest/download/nido-darwin-amd64 -o nido
chmod +x nido && sudo mv nido /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/Josepavese/nido/releases/latest/download/nido-darwin-arm64 -o nido
chmod +x nido && sudo mv nido /usr/local/bin/
```

#### Install QEMU (Required)

Nido needs QEMU to run VMs:

```bash
# Linux (Debian/Ubuntu)
sudo apt install qemu-system-x86 qemu-utils

# macOS
brew install qemu

# Windows
choco install qemu
```

> **💡 Tip:** Run `nido doctor` after installation to verify your setup!

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

# Image & Cache Management 🆕
nido images list                    # Browse cloud images (shows file sizes)
nido images pull ubuntu:24.04       # Pull official Ubuntu 24.04 image
nido cache ls                       # List cached images and their sizes
nido cache info                     # Show cache stats (total size, age)
nido spawn my-vm --image ubuntu:24.04  # Spawn directly from any cloud image
nido spawn test-vm --no-cache       # Spawn without saving image to local cache
```

## Nido Flavours & Split Distribution 🧁📦

Nido Flavours are pre-built, optimized VM environments (like Lubuntu, XFCE, or specialized dev stacks) maintained by the community.

To bypass storage limits and ensure fast downloads:

- **High Compression:** All flavours are optimized for size before distribution.
- **Split Distribution:** Large images (>2GiB) are automatically distributed in segments via GitHub Releases.
- **Auto-Reassembly:** `nido` automatically handles multi-part downloads and reassembles images on the fly.

## Shell Completion 🆕

`nido` now supports full shell completion for **Bash** and **Zsh**. Never guess a command or template name again!

```bash
# Generate completion script
nido completion bash > ~/.nido/bash_completion
echo "source ~/.nido/bash_completion" >> ~/.bashrc
```

## Commands

| Command | What it does | Example |
| :--- | :--- | :--- |
| `spawn <name> [--image <img> --gui]` | Create and start a VM with optional GUI | `nido spawn vm1 --image xfce:24.04 --gui` |
| `start <name> [--gui]` | Revive a stopped VM with optional GUI | `nido start test-vm --gui` |
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
| `cache ls` | List cached images | `nido cache ls` |
| `cache info` | Show cache statistics | `nido cache info` |
| `cache prune` | Remove all cached images | `nido cache prune` |

### GUI Support (VNC)

Nido can expose a graphical interface for VMs:

- Use the `--gui` flag with `spawn` or `start`.
- Run `nido info <name>` to see the VNC endpoint.
- Connect with any VNC client to `127.0.0.1:5900X`.

### Automation

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
| :--- | :---: | :---: | :---: | :---: |
| **AI Integration** | ✅ MCP | ❌ | ❌ | ✅ Cloud |
| **Local-first** | ✅ | ✅ | ✅ | ❌ |
| **Storage** | Compressed | Full boxes | Full images | Cloud |
| **Cross-Platform** | ✅ | ✅ | ✅ | ✅ |
| **Simplicity** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Humor Level** | 🐣🐣🐣 | 😐 | 😐 | 🤖 |

## Roadmap

The single source of truth lives here: `docs/ROADMAP.md`.

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
