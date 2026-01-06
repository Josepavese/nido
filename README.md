# 🪺 nido

Hatch a VM, do the work, let it fly away. Your AI agents will feel at home.

Nido is a **fast, lightweight, AI-first VM automation tool**.

You install it in seconds, get a clean CLI, and AI agents can start spawning **real virtual machines** immediately.
No clusters. No dashboards. No babysitting.

Built on QEMU and hardware acceleration, Nido feels like a small CLI —
but it gives agents what containers can’t: **a full operating system, on demand**.

Think of it as a nest 🪺 for automation:
machines hatch 🐣, do their job, and fly away 🐦.

If your AI workflows need a real OS with zero friction, Nido stays out of the way and lets them run.

---

## Use case: hatch, fix, fly away

Say a CI agent needs a clean OS to reproduce a bug, apply a patch, and leave nothing behind.  
With Nido, it hatches a VM from an image **even if it isn't local yet**, does the job, and lets it fly away. 🐣🐦

```bash
# 1) See what's in the nest (no download yet)
nido images list

# 2) Hatch fast: Nido pulls the image on the fly if it's not cached
nido spawn bugfix-vm --image ubuntu:24.04

# 3) (optional) hop in and do the work
nido ssh bugfix-vm

# 4) Cleanup at light speed
nido delete bugfix-vm
```

Bottom line: a real OS as an execution surface for agents, minus the babysitting.

## Philosophy

**Automation first. Agents first. Speed first.**

- ⚡ **Fast by default**  
  Install fast, spawn fast, clean up fast. Heavy workflows are a bug.

- 🤖 **AI-first, not human-first**  
  The CLI is friendly, but the real contract is machine-to-machine.

- 🐣 **Ephemeral is the happy path**  
  VMs hatch, run, and disappear. Cleanup is part of the lifecycle.

- 🪺 **Workflows over infrastructure**  
  Agent → environment → action → result → cleanup.  
  The VM is just the execution surface.

- 🧠 **Small and opinionated**  
  No cloud cosplay. No feature bloat. Strong defaults only.

---

## Features

- ⚡ **Fast & lightweight** — install in seconds, no background services  
- 🐣 **Real VMs on demand** — full OS via QEMU, hardware acceleration when available  
- 🤖 **Built for AI agents** — clean CLI + native MCP server  
- 🧺 **Templates** — reproducible environments, no interactive provisioning  
- 🧹 **Automatic cleanup** — create → run → destroy, crash-safe  
- 🪶 **Cross-platform & local-first** — Linux, macOS, Windows

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

- **Zero-Touch Registry:** Flavours are automatically discovered from GitHub Releases using our smart scanning strategy. No manual registration needed. 🤖
- **Split Distribution:** Large images (>2GiB) are automatically distributed in segments via GitHub Releases to ensure high availability.
- **Auto-Reassembly:** `nido` handles multi-part downloads and reassembles images on the fly.
- **Trusted Provenance:** Recognized flavours are marked with a bold `[PRECONFIGURED]` badge in `nido image list`.

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
3. Follow the [tone of voice guidelines](docs/.tone_of_voice.md) (keep it fun!)

## License

MIT License - See [LICENSE](LICENSE) for details.

---

<p align="center">
  <i>Made with ❤️ for the Agentic future.</i><br>
  <i>"It's not a VM, it's a lifestyle."</i> 🪺
</p>
