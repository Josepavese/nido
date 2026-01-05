# Changelog

All notable changes to Nido will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
