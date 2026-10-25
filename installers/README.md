# Install Nido

[Documentation](../docs/README.md) · [Distribution channel status](../docs/DISTRIBUTION.md)

Choose a binary installer, a package manager or an MCP client integration.
The binary installers require no repository clone or Go toolchain. QEMU is a
separate runtime dependency.

AMD64 and ARM64 host/guest selection, requirements and data-preserving upgrades:
[Architecture guide](../docs/architectures.md). Omitted architecture selects native; use `--arch`
on CLI, `arch` on MCP, or the TUI **Arch** selector for an explicit guest ISA.

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
| [npm](https://www.npmjs.com/package/nido-mcp), Linux / macOS / Windows | `npm install -g nido-mcp` |
| npm without global install | `npx -y nido-mcp version` for the CLI; `npx -y nido-mcp mcp` for an MCP client |
| Homebrew, macOS / Linux | `brew install Josepavese/nido/nido` |
| Scoop, Windows | Add `https://github.com/Josepavese/scoop-nido` as bucket `nido`, then `scoop install nido` |
| WinGet, Windows | `Josepavese.Nido`; [check availability below](#winget-windows) while Microsoft reviews the initial package |
| APT, Debian / Ubuntu | [Nido APT repository](#apt-debian--ubuntu) → `sudo apt-get install nido` |
| DNF, Fedora / RHEL | [Nido DNF repository](#dnf-fedora--rhel) → `sudo dnf install nido` |
| Container, Docker / Podman | `docker run -i --rm --device /dev/kvm --group-add "$(stat -c %g /dev/kvm)" -v nido-home:/home/nido/.nido ghcr.io/josepavese/nido mcp` |
| Go toolchain | `go install github.com/Josepavese/nido/v4/cmd/nido@latest` |
| Nix flake (Linux / macOS) | `nix run github:Josepavese/nido` (builds from source, wraps QEMU) |

The npm launcher downloads the matching binary on first use and verifies it
against release `SHA256SUMS`. Go install builds from source and does not install
QEMU. Companion package channels can lag a new GitHub release; availability and
credential-dependent channels are tracked in [distribution](../docs/DISTRIBUTION.md).

### Container image

`ghcr.io/josepavese/nido` is published from the release assets on every tag and
bundles QEMU. It is the quickest way to hand a disposable VM manager to an agent
that already runs in a container:

```bash
docker run -i --rm --device /dev/kvm \
  --group-add "$(stat -c %g /dev/kvm)" \
  -v nido-home:/home/nido/.nido \
  ghcr.io/josepavese/nido mcp
```

Without `--device /dev/kvm` and the matching group id the container falls back
to emulation or fails `nido doctor`; the image runs as the non-root `nido` user
with `/home/nido/.nido` as its data volume.

### Nix (flake, from source)

The repository ships a `flake.nix`; Nix builds the current source and wraps
`PATH` so the flake's QEMU is available at runtime:

```bash
nix run github:Josepavese/nido -- version
nix run github:Josepavese/nido -- doctor
```

`nix profile install github:Josepavese/nido` adds `nido` and `nido-validator`
to the profile. The version reported by the flake is read from `CHANGELOG.md`,
so it tracks tags without a manual bump. Nixpkgs itself is intentionally not
targeted: one flake avoids the weeks-long nixpkgs review while giving the same
`nix run` entry point.

### APT (Debian / Ubuntu)

The release workflow publishes an APT repository on GitHub Pages, rebuilt from
the release binaries on every tag:

```bash
sudo install -d -m 0755 /etc/apt/keyrings
sudo curl -fsSL https://josepavese.github.io/nido/nido-archive-keyring.asc \
  -o /etc/apt/keyrings/nido.asc
echo "deb [signed-by=/etc/apt/keyrings/nido.asc] https://josepavese.github.io/nido/apt/ stable main" \
  | sudo tee /etc/apt/sources.list.d/nido.list
sudo apt-get update
sudo apt-get install nido
```

Packages exist for `amd64` and `arm64`. QEMU is a host prerequisite and is not
pulled in by the package: `sudo apt-get install qemu-system-x86 qemu-utils`,
then `nido doctor`. The APT `Release`/`InRelease` metadata is GPG-signed by the
Nido package archive key (`DBB87CB63A94ABA50B33C7B919C6A95C9D280B64`, expires
2029-10-24); the keyring is published at
`https://josepavese.github.io/nido/nido-archive-keyring.asc` and mirrored in the
repository at `distribution/packages/nido-archive-keyring.asc`.

### DNF (Fedora / RHEL)

```bash
sudo curl -fsSL https://josepavese.github.io/nido/rpm/nido.repo \
  -o /etc/yum.repos.d/nido.repo
sudo dnf install nido
```

Packages exist for `x86_64` and `aarch64`. Both the RPM packages and the DNF
`repomd.xml` are GPG-signed, and `nido.repo` sets `gpgcheck=1`,
`repo_gpgcheck=1` and `gpgkey=` to the published keyring, so DNF imports the key
automatically. Install QEMU first (`sudo dnf install qemu-kvm qemu-img`) and run
`nido doctor` afterwards.

### WinGet (Windows)

The `Josepavese.Nido` package has been submitted to the community repository:
[Microsoft review](https://github.com/microsoft/winget-pkgs/pull/460843).
The CLA and package validation have passed; initial publication still requires
moderator approval. Check whether the package has reached your WinGet catalog:

```powershell
winget search --id Josepavese.Nido --exact
```

Once it appears, install or upgrade it with:

```powershell
winget install --id Josepavese.Nido --exact
winget upgrade --id Josepavese.Nido --exact
```

While that review is pending, use the PowerShell installer, Scoop or npm.
QEMU remains a host prerequisite, whichever Nido channel you choose.

## MCP catalogs and client installation

| Method | Where to find Nido | How it runs |
| --- | --- | --- |
| [Official MCP Registry](https://registry.modelcontextprotocol.io/v0.1/servers/io.github.Josepavese%2Fnido/versions/latest) | Search for `io.github.Josepavese/nido` in a registry-aware client | The registry advertises the `nido-mcp` npm package and MCPB release bundle |
| [Glama](https://glama.ai/mcp/servers/Josepavese/nido) | Nido / `nido-mcp` by `Josepavese` | Discovery listing with instructions to install Nido and run MCP locally |
| [npm](https://www.npmjs.com/package/nido-mcp) | `nido-mcp` | Configure the client to launch `npx -y nido-mcp mcp` |
| [MCPB bundle](https://github.com/Josepavese/nido/releases/latest/download/nido.mcpb) | `nido.mcpb` in every GitHub release | Import into a client that supports MCPB; QEMU must be available on the same host |

These catalogs help clients discover the server. Nido runs locally over stdio;
QEMU, VM disks and host permissions belong to the machine running that process.
Use the [MCP guide](../docs/mcp.md#configure-a-client) for the exact configuration.
On Glama, follow the local installation instructions. Its hosted Deploy action
starts a separate environment and cannot manage the VMs on your computer.
Discovery integrations (Glama promotion, directory listings) have their own activation
status in the [distribution guide](../docs/DISTRIBUTION.md).

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
Self-update replaces the executable and synchronizes the bundled catalog; it
does not delete VM disks, seeds, configuration or runtime metadata. From 4.10.2,
archive transfer has a 15-minute total budget, while each metadata request and
response headers remain bounded to 30 seconds. Prematurely disconnected bodies
are retried up to three attempts with HTTP Range resumption; a server that
ignores Range is handled by restarting the download safely. Cancellation or
failure leaves the previous executable installed and removes private staging.
Partial downloads are resumed within the same update invocation only.

An older executable still uses its original updater: publishing a new release
cannot change that timeout in place. If it times out, use the original package
manager or rerun the binary installer above, preserving the same Nido data
directory. Never uninstall to repair a download timeout. When crossing from the
previous AMD64-only metadata to 4.10, follow the explicit
[architecture migration](../docs/architectures.md) before reopening the fleet;
the conversion preserves existing VM disks and backs up original metadata.

Read the [changelog](../CHANGELOG.md) for upgrade notes, especially host PCI
recovery records. Reconnect MCP clients after an executable upgrade so they
load the current process and schemas.

`nido config --json` shows effective storage locations. Moving binaries or changing
storage settings does not migrate existing VM disks. Preserve useful disks,
templates, configuration and recovery records; do not treat them as installer
scratch. See [operations and storage](../docs/usage.md#isolation-and-storage).
