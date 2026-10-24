# Install Nido

[Documentation](../docs/README.md) · [Distribution channel status](../docs/DISTRIBUTION.md)

The binary installer is the simplest starting point. It requires no repository
clone or Go toolchain. QEMU is a separate runtime dependency.

## Binary installers

Linux / macOS:

```bash
curl -fsSL https://github.com/Josepavese/nido/releases/latest/download/install.sh | bash
```

Windows PowerShell:

```powershell
irm https://github.com/Josepavese/nido/releases/latest/download/install.ps1 | iex
```

Open a new terminal so PATH changes take effect:

```bash
nido version
nido doctor
```

The installers place the release in your user-level `.nido` directory and install
the bundled registry. They check QEMU prerequisites and offer installation where
supported. On Windows the installer can register/install winget, install QEMU
and offer to enable Windows Hypervisor Platform. Enabling WHPX requires local
administrator approval and a Windows restart. See [Windows support](../docs/windows.md).

If you prefer to inspect the installer first, download `install.sh` or
`install.ps1` from the [release assets](https://github.com/Josepavese/nido/releases/latest),
review it, then execute the saved file.

## Package managers

| Channel | Command |
| --- | --- |
| Homebrew, macOS / Linux | `brew install Josepavese/nido/nido` |
| Scoop, Windows | Add `https://github.com/Josepavese/scoop-nido` as bucket `nido`, then `scoop install nido` |
| npm launcher | `npm install -g nido-mcp` |
| npm without global install | `npx -y nido-mcp mcp` |
| Go toolchain | `go install github.com/Josepavese/nido/v4/cmd/nido@latest` |

The npm launcher downloads the matching binary on first use and verifies it
against release `SHA256SUMS`. Go install builds from source and does not install
QEMU. Companion package channels can lag a new GitHub release; availability and
credential-dependent channels are tracked in [distribution](../docs/DISTRIBUTION.md).

## Host prerequisites

- `qemu-system-*` and `qemu-img` appropriate for the host and guest architecture.
- An SSH client for guest connections.
- Enough disk space for base downloads, transfer staging and growing VM disks;
  enough RAM for the host and every running guest.

Linux can use KVM when available and accessible; macOS can use HVF; Windows can
use WHPX. Unsupported or unavailable acceleration falls back to software
emulation where applicable. `nido doctor` reports the effective capability.
Hardware acceleration is not guaranteed inside another VM or a hosted CI runner.
ARM Linux / Termux setups need compatible QEMU packages; do not assume that a
binary for the CPU alone establishes guest or Android runtime compatibility.

Example QEMU installation commands:

```bash
# Debian / Ubuntu
sudo apt install qemu-system-x86 qemu-utils

# macOS
brew install qemu
```

For Windows, use the quick installer's supported setup or the options linked
from [QEMU downloads](https://www.qemu.org/download/#windows).

## Source builds

The canonical Go toolchain line is in [`.go-version`](../.go-version); use its
current secure patch release. The lightweight source installer downloads a source
archive instead of cloning the repository:

```bash
curl -fsSL https://raw.githubusercontent.com/Josepavese/nido/main/installers/build-from-source.sh | bash
```

Contributors can follow [Contributing](../CONTRIBUTING.md) for builds outside the
checkout. `bin/install-from-source.sh` is the full source-install path; inspect
its dependency and installation behavior before running it.

## Upgrades and existing data

Use the package manager that installed Nido, or `nido update` for the standalone
binary updater. Do not uninstall first: uninstall removes Nido data.
Read the [changelog](../CHANGELOG.md) for upgrade notes, especially host PCI
recovery records. Reconnect MCP clients after an executable upgrade so they
load the current process and schemas.

`nido config --json` shows effective storage locations. Moving binaries or changing
storage settings does not migrate existing VM disks. Preserve useful disks,
templates, configuration and recovery records; do not treat them as installer
scratch. See [operations and storage](../docs/usage.md#isolation-and-storage).
