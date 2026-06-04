# Changelog

All notable changes to Nido will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [4.5.26] - 2026-06-04 "The Scheduled Gate"

### Fixed

- CI now uses Go `1.26.4`, resolving the `govulncheck` failures caused by standard-library vulnerabilities in Go `1.26.3`.
- Registry update PR checks are no longer blocked by the stale Go toolchain used after `v4.5.25`.

### Changed

- The release workflow now dispatches every scheduled workflow after publication and waits for the workflow result.
- When a scheduled workflow manages a bot PR, the release workflow also waits for that PR's status checks so downstream automation failures are visible during release validation.
- Scheduled workflows are required to expose `workflow_dispatch`, making them manually and release-verifiable.

### Tested

- `GOTOOLCHAIN=go1.26.4 go test ./...`.
- `GOTOOLCHAIN=go1.26.4 make lint`.
- `GOTOOLCHAIN=go1.26.4 make build`.
- `bash -n bin/verify-scheduled-workflows.sh`.

## [4.5.25] - 2026-06-02 "The MCP Parity Release"

### Added

- MCP system coverage now includes version, update checks, update, global config set, accelerator list, MCP registration, shell completion generation, image build aliasing, and guarded uninstall.
- MCP resources now expose system version, accelerator discovery, and MCP registration data for read-only agent inspection.

### Fixed

- MCP VM creation now matches the CLI spawn path for `web`/`ftp` defaults and local images produced by blueprints, including Windows blueprint SSH metadata and seed files.
- Global config key validation is shared between CLI and MCP to avoid drift.

### Tested

- Linux test suite: `go test ./...`.
- Release lint/build checks: `make lint`, `make build`.
- MCP smoke tests against both `./bin/nido mcp` and `/home/jose/.nido/bin/nido mcp` confirmed 11 system actions.
- Local install doctor passed 9/9 checks.

## [4.5.24] - 2026-06-02 "The Self-Syncing Registry"

### Fixed

- `nido update` now syncs bundled registry assets from the release archive into the PAL home, keeping installed blueprints aligned with the updated binary.
- Blueprint commands now auto-sync the bundled registry embedded in the binary before resolving blueprints, repairing PAL homes updated by older updater versions.
- Registry sync creates a backup before overwriting bundled files and skips repeated backups when the installed registry is already current.

### Tested

- Linux test suite: `go test ./...`.
- Release lint/build checks: `make lint`, `make build`.
- Isolated PAL-home smoke confirmed `nido blueprint info windows-11-iot-ltsc-eval --json` repairs stale Windows blueprint assets and `nido build windows-11-iot-ltsc-eval --json` resolves the fixed blueprint.

## [4.5.22] - 2026-05-14 "The Safer Windows Blueprint"

### Fixed

- Windows blueprints now install and enable OpenSSH Server during the elevated `specialize` phase instead of relying on first-login user commands.
- `nido-validator` now uses direct SSH readiness checks, safer Windows subprocess timeout handling, and strict validator-owned resource names before cleanup.
- Validator cleanup and prune no longer target user VMs such as `nido-win-pal` or legacy non-validator names.

### Tested

- Linux test suite: `go test ./...`.
- Validator safety smoke with `nido-win-pal` present confirmed prune is skipped instead of deleting non-validator VMs.

## [4.5.21] - 2026-05-13 "The Windows Blueprint" 🪟🧬

### Added

- Windows image blueprints for Windows 11 Enterprise Evaluation, Windows 11 IoT Enterprise LTSC 2024 Evaluation, and Windows Server 2022 Evaluation (Server Core).
- CLI, TUI, MCP, JSON output, and cache integration for buildable blueprints.
- Built-in Go seed ISO writer, removing the runtime dependency on mkisofs/genisoimage/xorriso for cloud-init and blueprint automation.

### Improved

- Windows host PAL support for QEMU lifecycle management, including detached process launch, stop/delete handling, SSH command mode, and WHPX-to-TCG fallback.
- Windows installer dependency flow: winget registration/install, QEMU install via winget, and optional Windows Hypervisor Platform enablement.
- Registry ordering now shows Nido flavours first, blueprints second, and official cloud images last.

### Tested

- Windows host smoke test on a real Windows VM: installer script parsing, `nido doctor`, catalog/blueprint listing, and basic spawn/SSH/stop/delete lifecycle.
- Linux test suite: `go test ./...`.

## [4.5.2] - 2026-01-20 "The Clean Break" 🧼📺✨

### Fixed 🐛

- **Hatchery TUI**: Resolved focus synchronization issue where list selection didn't update after pulling a new image.
- **Registry & Catalog**: Fixed duplicate image entries caused by dirty local caches; implemented auto-deduplication.
- **Port Allocation**: Fixed collision issue where stopped VMs' reserved ports were ignored during new spawns.
- **Validator**: Corrected `default` workflow to stop assuming a default template exists (bootstraps from base image now).

### Changed 🔧

- **Legacy Cleanup**: Vaporized `TemplateDefault`, `TUI.FooterLink`, and `TUI.TabLabels` from the codebase. `nido spawn` now demands explicit intent (image or template) instead of failing on a phantom default.
- **Error Feedback**: Prominent modals for destructive failures (Template/Image deletion) are now standard.

## [4.5.1] - 2026-01-19 "The Resilient Nest" 🛡️🩹🐣

### 🚀 Improved

- **Interactive Installers**: Fixed a critical bug where interactive prompts (like the KVM permission fix) failed when the installer was piped (e.g., `curl | bash`). Redirection to `/dev/tty` now ensures 100% human-interactive success.
- **Enhanced OOM Diagnostics**: The GUI now detects memory exhaustion errors (`Cannot allocate memory`) and provides a dedicated diagnostic modal (ERR_MEM 🧠) with hardware optimization advice.
- **Vocal KVM Checks**: Both the installer and `nido doctor` are now more verbose and explicit about the mandatory session restart (or `newgrp` usage) required after applying virtualization permissions.

## [4.5.0] - 2026-01-19 "The Nested Nest" 🪆🪺🧪

### 🚀 Improved

- **Nested Virtualization Hardening**: The installers and `nido doctor` now detect "Permission denied" errors on `/dev/kvm`. They proactively offer to add your user to the `kvm` group, resolving the most common blocker for nested VM environments. Zero friction, total power. ⚡🛡️
- **Smarter Diagnostics**: `nido doctor` now checks specifically for read/write access to hypervisor modules, not just their existence.

## [4.4.9] - 2026-01-19 "The Universal Nest" 🐣🌏🔋

### 🚀 Improved

- **Universal Proactive Install**: Windows users now get the same proactive treatment as Linux/Mac. `quick-install.ps1` now detects missing QEMU and offers to install it via `winget`. Onboarding is now truly zero-friction across all dimensions of the multiverse. 🌌

## [4.4.8] - 2026-01-19 "The Proactive Nest" 🐣🔋

### 🚀 Improved

- **Interactive Proactive Install**: The installers (`quick-install.sh` and `install-from-source.sh`) now detect missing QEMU dependencies and offer an "Easy Mode" to automatically install them for you. Onboarding is now zero-friction. ⚡

## [4.4.7] - 2026-01-19 "The Actionable Nest" 🛠️🪺

### 🚀 Improved

- **Robust Quick-Install**: The installer is now architecture-aware, recommending `qemu-system-arm` on ARM64 and automatically including `sudo apt update` to prevent repository sync failures.
- **Actionable Doctor**: `nido doctor` now provides direct, copy-pasteable installation commands when QEMU or KVM are missing.

## [4.4.6] - 2026-01-19 "The Final Descent" 💀🪺

### 🔧 Fixes & Polish

- **Absolute Self-Destruct:** Fixed a critical message-type mismatch where the `tea.Quit` signal was being sent as a function reference instead of a realized message. The TUI now terminates with surgical precision after an uninstallation. 🏁
- **Genome Hardening:** Final cleanup of redundant lifecycle interceptors.

## [4.4.5] - 2026-01-19 "The Stoic Nest" 🏺🪺

### 🔧 Fixes & Polish

- **Glitch-Free Navigation:** Fixed an issue where the "Up to Date" modal would erroneously trigger during automatic background checks when navigating to the Evolution page. Modals now only appear during manual, user-initiated scans. 🖱️🚫
- **Quiet Evolution:** Background version checks are now silent unless an actual ascension is required.

## [4.4.4] - 2026-01-19 "The Responsive Nest" 🕹️🪺

### 🔧 Fixes & Polish

- **Sanitized Self-Destruct:** The GUI now correctly terminates after a `self-destruct` operation. The application successfully shuts down once the uninstallation sequence is complete. 🧹
- **Evolutionary Feedback:** Added explicit feedback when checking for updates. If your Nest is already at the latest evolutionary state, a modal now confirms that no further ascension is required. 🧬
- **Life-Cycle Synchronization:** Improved internal message propagation to ensure and state changes are correctly reflected across the TUI.

## [4.4.3] - 2026-01-19 "The Turbo Nest" 🚀🪺

### ✨ New Toys

- **Ultra-Fast Decompression:** Migrated all Nido flavours to Zstandard (Zstd). Your agents can now hatch faster than ever with superior compression ratios. 📼
- **ComfyUI Flavour:** Added the `ubuntu-24.04-comfyui` flavour to the official registry. A heavy-duty stable diffusion environment, pre-optimized and split for smooth delivery. 🎨
- **Enhanced Viewport:** Expanded the arcade viewport dimensions to **84x26**. More columns for your terminal data, more rows for your agent's logs. 📺

### 🔧 Fixes & Polish

- **Genetic Path Expansion:** Repaired a glitch in the matrix where `${HOME}` in `config.env` wasn't being correctly synthesized. Environment variables are now fully expanded in your configuration paths. 🧬
- **Sanitized Cleanup:** The `uninstall` sequence now surgically removes Nido entries from your shell configuration files ($HOME/.bashrc, $HOME/.zshrc), ensuring a clean exit. 🧹
- **Documentation Sync:** Harmonized the README with the modern installer behavior. No more hunting for the non-existent `~/.nido/env` file. 🕹️

## [4.4.2] - 2026-01-18 "The Bulletproof Nest" 🛡️🪺

### ✨ New Toys

- **Nido Validator:** A new CLI tool `nido-validator` for contributors. It puts the nest through a rigorous bootcamp, verifying CLI commands and MCP protocols. Perfect for ensuring your code doesn't break the matrix. 🏋️‍♂️

### 🔧 Fixes & Polish

- **Evolution Protocol Repaired:** The `update` command now correctly identifies empty releases and reports errors with surgical precision ("Binary not found" vs "exit status 1"). 🏥
- **Pipeline Hardening:** Our release droids now refuse to ship empty crates. The CI workflow explicitly verifies asset existence before publishing. 🤖
- **TUI Synchronization:**
  - **ASCII Validation:** Swapped emojis for rock-solid ASCII `[!]` indicators.
  - **Focus Logic:** `Tab` navigation now dives deep into nested rows (like port forwarding controls).
  - **Layout:** Reverted specific config pages to 50/50 splits for visual harmony.

## [4.4.1] - 2026-01-18 "The Polished Nest" 🧼🪺

### 🎉 Major Features

- **Desktop Integration:** Native launcher icons and shortcuts for Linux (with improved ~/.desktop entries), macOS (App Bundles), and Windows (Start Menu). Use `nido` like a real desktop app! 🖥️
- **Uninstall Protocol:** Added `nido uninstall`. A "nuclear option" that cleanly removes the binary, data directory (~/.nido), shell config lines, and all desktop shortcuts. 🧹
- **New Iconography:** Default icon updated to an 80s-themed pixel art CRT nest. Retains the "nerdy" soul of the project. 🕹️

### 🔧 Improvements

- **TUI Synchronization:** Merged experimental TUI features relative to configuration pages and modal styling.
- **Housekeeping:** Removed legacy debug/repro scripts that cluttered the repo.

## [4.4.0] - 2026-01-16 "The Arcade Nest" 🕹️🐦

### 🎉 Major Features

- **80s Arcade TUI:** The "About" page has been completely reimagined as a retro arcade cabinet. Validated with empirical pixel-perfect rendering. Includes a high-score table ("HALL OF FLOCK") and blinking "INSERT COIN" text. Pure nostalgia. 👾
- **Idempotent Navigation:** Fixed a critical issue where the TUI shell could spawn duplicate navigation tabs ("piolotto spam"). The wiring is now rock-solid.
- **TUI Focus Logic:** Global improvements to focus handling, ensuring `Esc` and `q` behave predictably across all modals and views.

### 🔧 Quality of Life

- **ASCII Art Stability:** The Nido logo in the TUI is now geometrically stabilized (25x4 chars) to prevent visual drift on resize.
- **Sidebar Restoration:** Fixed truncation issues in the sidebar to ensure all VM details are visible.
- **Real-time Input Filtering:** Inputs in the TUI now strictly filter invalid characters (like spaces in VM names) in real-time.

## [4.3.4] - 2026-01-08

### ✨ UI & DX Refinements

- **Fleet Speed Hatch:** Added a prominent blue `[⊕] SPEED HATCH` button at the bottom of the Fleet view. This serves as a quick shortcut to the Hatchery, accessible via keyboard (arrow down from list) or mouse. 🛸
- **TUI Focus Logic:** Improved the Fleet view to manage focus between the VM list and action buttons.

## [4.3.3] - 2026-01-08

## [4.3.2] - 2026-01-08

### 🐛 Bug Fixes

- **README:** Deduplicated "Configuration" headers for improved documentation structure and navigation.

## [4.3.1] - 2026-01-08

### 🐛 Bug Fixes

- **README:** Fixed a markdown formatting error in the Usage section that caused subsequent text to be rendered incorrectly.

## [4.3.0] - 2026-01-08 "The Elegant Nest" 🪺✨

### 🎉 Major Features

- **Killer Feature: Linked Clones:** Refactored image caching into a proper "Linked Clones" system. VMs now use QCOW2 backing files for near-instant spawning and massive space savings. 🧬
- **Smart Cache Protection:** Implemented intelligent protection that prevents base images from being deleted if any VM is currently using them as a backing file. 🛡️
- **Zero-Config Source Cycling:** In the TUI Hatchery, users can now cycle through available image/template sources using **Left/Right arrow keys** when the source field is focused. ⌨️
- **Integrated Help System:** Added `nido help` command for high-level guidance directly in the terminal.
- **Structured JSON Output:** Added `--json` support for automation and GUI integrations on core commands (ls, info, spawn, start, stop, delete, prune, template, images, cache, version). 🪺
- **Full JSON Coverage:** Added JSON output for `doctor`, `config`, and `register`.

### 🎨 TUI & UX Refinements

- **Visual Cues:** Destructive actions (Kill, Delete) in the Fleet View are now styled in aggressive red. 🔴
- **State Management:** Fixed selection persistent and detail pane clearing after VM deletion.
- **Hatchery UI Polish:** Removed confusing placeholders and fixed arrow-key conflicts in the spawn form.

### 🔧 CLI & Maintenance

- **Conflicting Flag Consolidation:** Deprecated `CACHE_IMAGES` in favor of `LINKED_CLONES` (with backward compatibility) to resolve download redundancy.
- **Shell Completion v2:** Suggestions are now fully synchronized with the 4.3.0 feature set, including `help` and `LINKED_CLONES` configuration.
- **Codebase Purification:** Decimated legacy binaries, redundant temporary artifacts, and dead code paths. 🧹
- **Narrative Documentation:** Standardized English code comments with a "Senior Engineer" tone, explaining the *why* behind the magic.
- **Improved UX Guidance:** CLI usage and completion now prominently highlight the `--json` option and supported commands.

### 🐛 Bug Fixes

- **QEMU Write‑Lock Fix:** Stop now waits for the real QEMU daemon PID (from the pidfile), preventing stale disk locks on restart. 🛠️
- **Idempotent Cleanup:** Delete, template delete, and cache remove now handle “already gone” cases gracefully (including JSON mode). 🧹

## [4.1.2] - 2026-01-07 "Genetic Pruning" 🧬✂️

### 🎉 Major Features

- **Extinction Protocol:** Added `nido template delete` and `vm_template_delete`. You can now remove templates from the graveyard. Use responsibly. 🦖

## [4.1.1] - 2026-01-07 "The Self-Ascending Nest" 🕊️✨

### 🎉 Major Features

- **Self-Update Protocol:** Introduced `nido update`. The nest can now scan for newer genetic sequences on GitHub and ascend to the latest evolutionary state automatically. 🤖
- **Automated Version Awareness:** `nido version` now performs a non-blocking check for updates and notifies you if a newer version is available.

### 🔧 CLI & UX Refinements

- **Consistent Aliases:** Fixed `nido images ls` (and `nido cache ls`) to align with top-level command patterns. UX friction reduced. 🛸
- **Dynamic Completions:** The `update` process now synchronizes your shell completion scripts automatically, ensuring new commands are always under your fingertips.
- **Better Guidance:** Updated `printUsage` to reflect the new system operations.

### 🐛 Bug Fixes

- **Doc Link Restoration:** Fixed the broken link to the Tone of Voice guidelines in the README.
- **Compiler Discipline:** Refactored completion logic to resolve undefined symbol errors during build.

## [4.1.0] - 2026-01-07 "The Autonomous Nest" 🪺✨

### 🎉 Major Features

- **Zero-Touch Flavour Discovery:** The `registry-builder` now scans GitHub Releases automatically. No more manual JSON edits when publishing new flavours. 🤖
- **Segmented Integrity:** Full support for multi-part downloads with automated `.sha256` verification. 🧩
- **Release-Driven Synchronization:** GitHub Actions now sync the registry in real-time whenever a new flavour is published or an existing one is updated.

### 🎨 UI & UX

- **Branded Provenance:** Nido Flavours are now grouped and highlighted with a bold `[PRECONFIGURED]` badge. Know exactly what's optimized for your agents.
- **Categorized Registry:** Clear separation in `nido image list` between official upstream proxies and Nido's premium environments.

### 🔧 Bug Fixes & Reliability

- **Hardened CirrOS Support:** Resolved fragile `cloud-init` issues by implementing a smart shell-script fallback for minimal metadata collectors.
- **Per-Image SSH Users:** Support for `ssh_user` overrides in the registry (e.g., `cirros` for CirrOS images).
- **Persistent State Management:** QEMU provider now ensures state directories exist before spawning, preventing silent status failures during first-time setup.
- **Better Diagnostics:** Added serial console logging to captured files for easier boot-time debugging.

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

[4.3.0]: https://github.com/Josepavese/nido/compare/v4.2.0...v4.3.0
[2.0.0]: https://github.com/Josepavese/nido/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/Josepavese/nido/releases/tag/v1.0.0
