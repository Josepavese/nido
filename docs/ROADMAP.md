# Nido roadmap

Nido focuses on local, agent-driven VM environments. This page separates shipped
capabilities from candidate work; it does not promise delivery dates.

## Available today

- QEMU VM lifecycle through CLI, TUI and a built-in stdio MCP server.
- SSH connection support, loopback port forwarding and optional guest display.
- Persistent VM resource configuration, raw QEMU argument replacement/clearing
  and supported host PCI passthrough behind HAL safety and ownership checks.
- Cloud-init seed generation and custom user-data.
- Cached cloud images, disk templates, QCOW2 linked clones and dependency-aware
  cache protection.
- Automated discovery of supported upstream cloud releases, checksum validation
  and buildable image blueprints, including Windows evaluation guests.
- Structured CLI JSON, generated help/completion, diagnostics and a verified
  standalone updater.
- Release archives, an MCP bundle, npm launcher and automated distribution
  channels. See [channel status](DISTRIBUTION.md) for what is actually live.

Host support and qualification are different claims. Linux, macOS and Windows
have native backends and cross-platform CI. Physical PCI qualification has a
specific device scope; Windows host smoke testing does not prove every intensive
workload. See [architecture](ARCHITECTURE.md) and [Windows](windows.md).

## Candidate work

These directions need design and acceptance criteria before implementation:

| Direction | Gap to resolve |
| --- | --- |
| First-boot feedback | Make guest readiness and provisioning failures easier to observe |
| Remote agent transport | Authentication, permissions and lifecycle for an HTTP/SSE service; current MCP is stdio only |
| Stronger long-run qualification | Repeated lifecycle loops, GUI sessions and workload-specific host testing |
| Flavour publishing experience | Keep recipes, large-asset distribution and reproducible builds easy to maintain |
| Snapshot / restore workflows | A defined state and data model beyond current disk templates |
| Fleet integrations | Concrete batch/CI consumers without adding a parallel VM engine |

Open a focused [issue](https://github.com/Josepavese/nido/issues) with the intended
user outcome and proof of success. Keep shipped behavior in the guides and
release history in the [changelog](../CHANGELOG.md), rather than leaving completed
features in a future-work list.
