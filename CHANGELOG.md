# Changelog

All notable changes to Nido will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [4.0.1] - 2026-01-06 "The Delivery Drone" 🛸

### 🔧 Maintenance & Reliability

- **Release Integrity Safeguards:**
  - New `bin/verify-release.sh` tool to ensure all binaries and assets are correctly published.
  - Hardened CI/CD workflows to prevent "partial" releases during manual interventions.
- **Fixed Installer Detection:** Resolved an issue where the `quick-install.sh` would occasionally see older versions due to release metadata delays.

### ✨ Maintainer Tools

- **Flavour Sync Tool:** Added `bin/upload-flavours.sh`, a professional, idempotent utility for maintainers to sync local image segments with GitHub Releases. It includes a real-time progress bar to watch those 1GB chunks fly. 🚀

## [4.0.0] - 2026-01-06 "Sentient Nest" 🤖

### 🎉 Major Features

- **AI-Ready Flavours:** Pre-configured VM environments with everything AI agents need
  - `lubuntu-24.04-minimal`: Lightweight desktop environment (1.2GB)
  - `lubuntu-24.04-xfce`: Full development environment with XFCE (1.8GB)
  - Both include: Python 3, Node.js, Git, Docker, development tools
  - Optimized with BleachBit and zero-fill for minimal size
  - Multi-part downloads (1GB chunks) for reliability

- **Cache Management System:** Full transparency and control over downloaded images
  - `nido cache ls` - List all cached images with sizes
  - `nido cache info` - Show cache statistics
  - `nido cache rm <image:version>` - Remove specific images
  - `nido cache prune [--unused]` - Clean up cache
  - New MCP tools: `vm_cache_list`, `vm_cache_info`, `vm_cache_remove`, `vm_cache_prune`

- **MCP Standardization (Breaking Change):** All MCP tools now use `vm_` prefix
  - Ensures consistency and discoverability for AI agents
  - 19 total tools, all properly namespaced
  - See "Breaking Changes" section below for migration guide

### Added

- **Lightweight Installation Options:**
  - `installers/quick-install.sh` - Download pre-compiled binary (~4MB)
  - `installers/quick-install.ps1` - Windows PowerShell installer
  - `installers/build-from-source.sh` - Minimal source download + build
  - One-liner installation: `curl | bash` or `irm | iex`

- **GUI Support (VNC):**
  - `--gui` flag for `spawn` and `start` commands
  - Automatic VNC port allocation and management
  - Display VNC endpoint in `nido info`
  - Full autocomplete support (Bash & Zsh)
  - MCP integration with `gui` parameter

- **Enhanced Autocomplete:**
  - Cache command completion (Bash & Zsh)
  - GUI flag completion
  - Improved subcommand suggestions

- **Developer Tools:**
  - `bin/publish-flavour.sh` - Automated flavour packaging and upload
  - `docs/MAINTAINERS.md` - Comprehensive maintainer guide
  - Multi-part download support in `downloader.go`

### Changed

- **Repository Structure:**
  - Moved installers to dedicated `installers/` directory
  - Renamed `bin/install.sh` to `bin/install-from-source.sh`
  - Organized flavour build scripts in `flavours/` directory

- **Naming Conventions:**
  - Flavours: `{OS}-{VERSION}-{FLAVOUR}` format
  - MCP tools: All prefixed with `vm_`

- **Installation Experience:**
  - Ultra-fast binary-only installation (no Git required)
  - Clear separation between end-user and developer workflows

### Fixed

- VNC test compilation errors in `qemu_test.go`
- Invalid schema validation test in `catalog_test.go`
- Autocomplete edge cases for cache subcommands

### Breaking Changes

> [!WARNING]
> **MCP Tool Renaming**: All MCP tools have been renamed with the `vm_` prefix. AI agents must update their tool calls.

**Migration Guide:**

| Old Name | New Name |
|----------|----------|
| `template_list` | `vm_template_list` |
| `template_create` | `vm_template_create` |
| `doctor` | `vm_doctor` |
| `config_get` | `vm_config_get` |
| `images_list` | `vm_images_list` |
| `images_pull` | `vm_images_pull` |
| `images_update` | `vm_images_update` |

**Why this change?**

- Consistent naming across all 19 MCP tools
- Better discoverability for AI agents
- Clear namespace separation

## [3.2.0] - 2026-01-06

### Added

- **Lightning-Fast Installation:** New `installers/` directory with `quick-install.sh` and `quick-install.ps1` that download only the pre-compiled binary (~4MB) instead of cloning the entire repository
- **Lightweight Source Build:** New `installers/build-from-source.sh` for tinkerers who want bleeding-edge features without cloning the full repository (downloads only essential source files)
- **Organized Structure:** Separated end-user installers (`installers/`) from developer tools (`bin/install-from-source.sh`)
- **GUI Support (VNC):** Added `--gui` flag to `spawn` and `start` commands for graphical desktop environments
- **VNC Port Management:** Automatic VNC port allocation and display in `nido info`
- **Nido Flavours:** Pre-built, optimized VM environments (Lubuntu Minimal, XFCE Development)
- **Multi-part Distribution:** Automatic handling of large image downloads split into 1GB chunks
- **Maintainer Documentation:** New `docs/MAINTAINERS.md` guide for repository management
- **Flavour Publishing Tool:** `bin/publish-flavour.sh` for automated flavour packaging
- **Comprehensive Test Coverage:** Added VNC-specific tests to `qemu_test.go`
- **Shell Autocomplete for GUI:** Added `--gui` flag completion for `spawn` and `start` commands in both Bash and Zsh
- **MCP GUI Support:** Exposed `gui` parameter in `vm_create` and `vm_start` MCP tools for AI agent integration

### Changed

- **Installation Method:** Users can now install with a single command (`curl | bash`) without Git
- **Repository Structure:** Moved flavour build scripts to dedicated `flavours/` directory
- **Naming Convention:** Standardized flavour names to `{OS}-{VERSION}-{FLAVOUR}` format
- **Binary Size:** Optimized to ~4.2MB for ultra-fast downloads

### Fixed

- Test compilation errors in `internal/provider/qemu_test.go` related to VNC port arguments

## [3.1.0] - 2026-01-05

### Added

- **Image Registry:** Built-in catalog of cloud images (Ubuntu, Debian, Alpine)
- `nido image list/pull` commands
- `nido spawn --image <name>` support
- `internal/image` package with downloader and verifier

## [3.0.0] - 2026-01-05

### Added

- Cross-platform QEMU support (Linux KVM, macOS HVF, Windows WHPX)
- MCP server with 12 tools for AI agent integration
- GitHub Actions CI/CD for automated testing
- Beta testing guide for macOS and Windows
- `nido start` command to revive stopped VMs
- `nido prune` command to remove all stopped VMs
- `config_get` MCP tool
- `vm_prune` MCP tool

### Changed

- **Rewrite:** Complete migration from Bash to Go
- Direct QEMU integration (no libvirt dependency)
- Port-based SSH forwarding instead of bridge networking
- Updated README with cross-platform installation instructions

### Removed

- Legacy Bash implementation (lib/*.sh)
- libvirt dependency
- Shell-based MCP server

## [2.0.0] - 2025-01-04

### Added

- Template compression system (.compact.qcow2)
- MCP server (stdio transport)
- Configuration management via config.env
- VM lifecycle commands (spawn, start, stop, delete)

### Changed

- Normalized command names (spawn/create/ls/delete/template)

## [1.0.0] - 2025-01-03

### Added

- Initial release
- Basic VM management with libvirt
- Template system
- SSH integration

---

[2.0.0]: https://github.com/Josepavese/nido/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/Josepavese/nido/releases/tag/v1.0.0
