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
- Shared local core events with an elected broker, native observers, live TUI
  refresh and opt-in MCP resource subscription/mute, shipped in v4.8.7.
  See [event qualification](events.md).
- Guest-requested shutdown (`nido shutdown`, `nido_vm action=shutdown`) that is
  distinct from forced `stop`, waits for the owned process, and never kills a
  guest that misses the timeout.
- Disk-only snapshots for stopped VMs (`nido snapshot`, `nido_vm
  snapshot_*`), with explicit running-VM rejection and no new resource/event
  architecture.

Host support and qualification are different claims. Linux, macOS and Windows
have native backends and cross-platform CI. Physical PCI qualification has a
specific device scope; Windows host smoke testing does not prove every intensive
workload. See [architecture](ARCHITECTURE.md) and [Windows](windows.md).

## Next steps and decisions

The directions below were evaluated against the current contracts and external
prior art. "Shipped" means the capability is present in the current tree and has
positive and negative proof; the other decisions record why work is not being
added now. They are not a promise of dates.

| Candidate | User outcome | Devil's advocate / decisive constraint | Decision |
| --- | --- | --- | --- |
| Guest-requested shutdown | Agents can stop a session after writing data without losing guest filesystem state; forced stop remains explicit | SSH is the only new dependency; guests without SSH cannot use it. Windows has no controllable QMP endpoint, and no channel can guarantee guest filesystem semantics after a delivered request times out. | **Shipped** as SSH guest poweroff with a bounded wait, no hidden force, and a documented `stop` fallback. Unix hosts add an ACPI/QMP fallback (`method=acpi`); Windows remains SSH-only by documented limitation. |
| Snapshot / restore | A cheap rollback point before a risky guest operation | Live RAM snapshots require QMP savevm/loadvm, a new state model and passthrough/device exclusions. Disk-only snapshots are simpler and match the current overlay architecture, but can be mistaken for full checkpoints. | **Shipped** as offline internal QCOW2 snapshots with explicit disk-only wording, stopped-VM guard, space-tag-safe parsing, per-VM busy locking and no auto-snapshot. Live/memory snapshots remain deferred. |
| Stronger long-run qualification | Confidence that repeated lifecycle, shutdown and snapshot flows do not drift | Adding more test loops can become a parallel test framework with no product consumer. | **Shipped for the current contract**: validator covers guest shutdown, ACPI fallback, snapshot lifecycle and running-VM rejection; the MCP durable scenario proves the stdio lifecycle end to end. Windows shutdown remains unqualified by documented platform limit. |
| Failed/timed-out guest shutdown | A caller can distinguish a confirmed guest poweroff from a forced process stop | A real timeout needs a guest that deliberately ignores both SSH and ACPI powerdown; the provider tests cover the delivered-timeout/no-fallback rule. | **Shipped**: the `shutdown-timeout` validator boots a real x86 guest with `-no-acpi`, masks SSH, proves `ERR_SHUTDOWN_FAILED` and a still-running VM, then cleans up with the explicit forced `stop`. |
| SIGINT mid-stop durability | A caller is not misled if the CLI is interrupted during a forced stop | SIGINT has no signal-safe guest flush barrier; adding one would change the host-process contract. | **Explicit limit with trigger**: docs state no durability guarantee; reopen only with a host interruption harness and a guest-sync requirement. Do not add a hidden grace period or retry. |
| Snapshot ENOSPC preflight/cleanup | A failed snapshot leaves no confusing partial state and names the space problem | Future guest writes determine internal-snapshot growth, so a fixed free-space threshold would reject valid rollback points and still not guarantee success. | **Error contract instead**: `qemu-img` remains authoritative, ENOSPC and image-lock failures are mapped, and the provider lease is retired on exit. Trigger: a reproducible partial snapshot on ENOSPC. |
| Remote agent transport (Streamable HTTP) | Clients on another machine can use one Nido server | Requires authentication, authorization, network exposure and a resident lifecycle; it contradicts the local stdio/no-daemon security model and creates a second control plane. | **Rejected for now.** Revisit only with a named authenticated consumer and a threat model. [MCP Streamable HTTP](https://modelcontextprotocol.info/specification/2026-07-28/basic/transports/streamable-http/) |
| MCP Tasks / async long operations | A client can disconnect and poll a long VM operation | The extension is new and client support is not universal; it needs a durable task store, TTL/cancellation semantics and a second result shape. Current wait is bounded and snapshots are offline and quick. | **Deferred.** Revisit when a real client requires tasks and a durable task owner is justified. [MCP Tasks](https://modelcontextprotocol.org/extensions/tasks/overview) |
| Fleet / CI batch integrations | Run many VMs with one lifecycle command | A batch API duplicates shell/CI orchestration and must define partial-failure, concurrency and resource ownership; the current one-VM contract is composable. | **Deferred** until a concrete CI consumer and its batch semantics are specified. |
| Event replay / durable event log | Reconstruct every historical change | Shared events are intentionally advisory; state files are authoritative. Persistent replay would add retention, delivery and migration problems and invite clients to treat events as the source of truth. | **Rejected.** Keep state-first recovery and explicit resource re-reads. |
| Guest exec / file transfer | Agents can run commands and move files without their own SSH client | `nido ssh` already composes the host SSH client; MCP deliberately does not expose arbitrary guest execution. A sync API would add permission and lifecycle boundaries without a proven gap. | **Rejected for now**; prefer the existing SSH contract. |
| Flavour publishing experience | Share prepared environments reproducibly | Registry, large-asset distribution and provenance are substantially larger than a VM lifecycle feature and have no agent-session consumer in this scope. | **Deferred.** |

External prior art used for the decisions, not copied as requirements:

- libvirt's separate graceful `shutdown` versus `destroy` semantics
  ([virsh shutdown](https://libvirt.org/manpages/virsh.html#shutdown)).
- QEMU's offline snapshot tooling and its warning against modifying an image in
  use ([qemu-img](https://www.qemu.org/docs/master/tools/qemu-img.html)).
- Agent-sandbox and microsandbox snapshot/suspend-resume direction
  ([Agent Sandbox snapshots](https://agent-sandbox.sigs.k8s.io/docs/sandbox/snapshots/),
  [microsandbox snapshots](https://docs.microsandbox.dev/sandboxes/snapshots)).
- MCP's Streamable HTTP transport and Tasks extension for the rejected/deferred
  remote and async directions (linked above).

Open a focused [issue](https://github.com/Josepavese/nido/issues) with the intended
user outcome and proof of success. Keep shipped behavior in the guides and
release history in the [changelog](../CHANGELOG.md), rather than leaving completed
features in a future-work list.
