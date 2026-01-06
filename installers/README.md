# Nido Installers 🪺

This directory contains installation scripts for end users.

## Quick Install (Recommended)

Download and install the pre-compiled binary. **No Git or build tools required.**

### Linux & macOS

```bash
curl -fsSL https://raw.githubusercontent.com/Josepavese/nido/main/installers/quick-install.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/Josepavese/nido/main/installers/quick-install.ps1 | iex
```

## Build from Source (Lightweight)

Want the latest code without cloning the whole nest? This script downloads only the essential source files and builds locally.

**Prerequisites:** Go 1.21+

```bash
curl -fsSL https://raw.githubusercontent.com/Josepavese/nido/main/installers/build-from-source.sh | bash
```

> **🐣 Why this option?** Perfect for tinkerers who want bleeding-edge features or need to customize the build, but don't want to download flavours, docs, and other non-essential files.

## Developer Install (Full Repository)

For contributors who want to build from source:

```bash
git clone https://github.com/Josepavese/nido
cd nido
bash bin/install.sh
```

---

## What Each Script Does

| Script | Purpose | Prerequisites | Target Users |
|--------|---------|---------------|--------------|
| `quick-install.sh` | Downloads pre-compiled binary from GitHub Releases | `curl` | End users (Linux/macOS) |
| `quick-install.ps1` | Downloads pre-compiled binary from GitHub Releases | PowerShell 5.1+ | End users (Windows) |
| `build-from-source.sh` | Downloads only source files and builds locally | Go 1.21+, `curl` | Tinkerers & power users |
| `../bin/install-from-source.sh` | Full repository clone and build | Go 1.21+, Git | Contributors & developers |

---

## Post-Installation

After installation, verify your setup:

```bash
nido version
nido doctor
```

Install QEMU if not already present:

```bash
# Linux (Debian/Ubuntu)
sudo apt install qemu-system-x86 qemu-utils

# macOS
brew install qemu

# Windows
choco install qemu
```

---

**"It's not a VM, it's a lifestyle."** 🪺
