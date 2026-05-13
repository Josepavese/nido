# Nido Installers 🪺

This directory contains installation scripts for end users.

## Quick Install (Recommended)

Download and install the pre-compiled binary. **No Git or build tools required.**

### Linux & macOS

```bash
{ curl -fsSL https://github.com/Josepavese/nido/releases/latest/download/install.sh || curl -fsSL https://raw.githubusercontent.com/Josepavese/nido/main/installers/quick-install.sh; } | bash
```

### Windows (PowerShell)

```powershell
try { irm https://github.com/Josepavese/nido/releases/latest/download/install.ps1 | iex } catch { irm https://raw.githubusercontent.com/Josepavese/nido/main/installers/quick-install.ps1 | iex }
```

## Build from Source (Lightweight)

Want the latest code without cloning the whole nest? This script downloads the GitHub source archive and builds locally.

**Prerequisites:** Go 1.26.3+

```bash
curl -fsSL https://raw.githubusercontent.com/Josepavese/nido/main/installers/build-from-source.sh | bash
```

> **🐣 Why this option?** Perfect for tinkerers who want bleeding-edge features or need to customize the build, but don't want to download flavours, docs, and other non-essential files.

## Developer Install (Full Repository)

For contributors who want to build from source:

```bash
git clone https://github.com/Josepavese/nido
cd nido
bash bin/install-from-source.sh
```

---

## What Each Script Does

| Script | Purpose | Prerequisites | Target Users |
|--------|---------|---------------|--------------|
| `quick-install.sh` | Downloads pre-compiled binary from GitHub Releases, checks QEMU | `curl` | End users (Linux/macOS) |
| `quick-install.ps1` | Downloads pre-compiled binary from GitHub Releases, checks QEMU/WHPX | PowerShell 5.1+ | End users (Windows) |
| `build-from-source.sh` | Downloads only source files and builds locally | Go 1.26.3+, `curl` | Tinkerers & power users |
| `../bin/install-from-source.sh` | Full repository clone and build | Go 1.26.3+, Git | Contributors & developers |

---

## Post-Installation

After installation, verify your setup:

```bash
nido version
nido doctor
```

The quick installers check runtime dependencies. On Windows, `quick-install.ps1` can register/install winget when needed, install QEMU via winget, and offer to enable Windows Hypervisor Platform for WHPX acceleration. Enabling WHPX requires administrator approval and a Windows restart.

Windows support is smoke-tested on a real Windows VM for installer parsing, diagnostics, catalog/blueprint listing, and basic VM lifecycle. It is usable for core workflows, but still needs heavier long-running validation.

If you prefer manual dependency installation:

```bash
# Linux (Debian/Ubuntu)
sudo apt install qemu-system-x86 qemu-utils

# macOS
brew install qemu

# Windows
winget install --id SoftwareFreedomConservancy.QEMU -e --source winget --scope machine --accept-package-agreements --accept-source-agreements
```

QEMU's official download page also links Windows alternatives, including Stefan Weil's installers and MSYS2 packages: <https://www.qemu.org/download/#windows>.

---

**"It's not a VM, it's a lifestyle."** 🪺
