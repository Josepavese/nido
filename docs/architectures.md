# AMD64 and ARM64 guests

Nido publishes native binaries for Linux, macOS and Windows on AMD64 and ARM64.
A 64-bit OS is required, including on Raspberry Pi. Installer selection concerns
Nido's executable; guest architecture is a separate, persistent choice.

## Choose a guest

Omitting `--arch` or passing `auto` chooses the physical host's native ISA for
catalog images and blank disks. Host detection accounts for Rosetta and Windows
application emulation. An unavailable native image is an error, never an implicit
foreign-architecture download. Ubuntu and Debian have both variants; the other
current catalog sources and shipped Windows blueprints are AMD64 only.

```bash
nido spawn native --image ubuntu:24.04
nido spawn arm-test --image debian:13 --arch arm64
nido spawn x86-test --image ubuntu:24.04 --arch amd64
nido images pull ubuntu:24.04 --arch arm64
nido images info ubuntu:24.04 --arch arm64 --json
nido info arm-test --json
nido ssh arm-test -- uname -m
```

The TUI Hatchery has an **Arch** selector: `auto`, `amd64`, `arm64`. Activate it
with Enter or Space. It indicates native acceleration or TCG when changed.
Registry rows show their ISA; pulling a row uses that exact variant. Fleet
details and structured CLI output expose the saved guest architecture.

MCP uses the same provider and remains a local stdio server:

```json
{"name":"nido_vm","arguments":{"action":"create","name":"arm-test","image":"ubuntu:24.04","arch":"arm64"}}
```

`nido_image` accepts `arch` for `info`, `pull`, `remove` and `cache_remove`.
Read `nido://image/ubuntu:24.04?arch=arm64` for a specific variant. Without the
query, the resource resolves natively. The catalog resource includes `arch`.
Architecture selection adds no unsolicited event subscriptions or notifications.

## Native virtualization and foreign emulation

| Host | Same guest ISA | Different guest ISA |
| --- | --- | --- |
| Linux AMD64 / ARM64 | KVM when available and permitted | QEMU TCG |
| macOS Intel / Apple Silicon | HVF when available | QEMU TCG |
| Windows AMD64 / ARM64 | WHPX when exposed by OS and QEMU | QEMU TCG |

Choosing the same ISA avoids instruction translation only when native
acceleration is actually available. A Raspberry Pi needs a 64-bit OS, a kernel
with KVM, accessible `/dev/kvm`, sufficient RAM and the ARM QEMU emulator.
Otherwise Nido selects TCG even for a native guest. Cross-ISA emulation is useful
for compatibility testing but can be substantially slower.

ARM64 uses QEMU's `virt` board, virtio PCI disks/networking and AArch64 UEFI.
Native ARM KVM selects the host GIC version, supporting hosts with GICv2 or
GICv3 without assuming a PC interrupt controller. Every VM owns writable UEFI variables; firmware code and the packaged variables
seed are never modified. ARM graphical guests use virtio GPU and USB input.
Private variable copies remain writable when the firmware seed lives in a
read-only package store. Version 4.10.1 also repairs an existing read-only
private variables file on start, preserving its contents.
See the upstream [virt board](https://www.qemu.org/docs/master/system/arm/virt)
and [WHPX requirements](https://www.qemu.org/docs/master/system/whpx.html).
Windows ARM WHPX requires Windows 11 24H2 with the 2025 API updates and a QEMU
build implementing ARM WHPX. Otherwise software emulation remains available.

Install the emulator for each guest ISA you intend to use. Debian/Ubuntu:

```bash
sudo apt install qemu-system-x86 qemu-system-arm qemu-efi-aarch64 qemu-utils
```

Homebrew QEMU bundles both emulators and EDK2. Windows needs a QEMU distribution
providing the chosen emulator and ARM EDK2 code/variables. Nido reports missing
firmware explicitly. `nido doctor` checks the host-native emulator and firmware;
starting a foreign guest validates that guest's dependencies separately.
`virtualization_capability` in VM info describes host capability, not a runtime
measurement or a guest-readiness guarantee. Verify guest readiness through SSH
and workload checks; a running QEMU process alone is insufficient.

## Disks, templates and upgrades

Catalog cache identity includes ISA: `ubuntu-24.04-arm64.qcow2`. An adjacent
`.nido.json` declares the architecture. Templates carry the same metadata and,
for ARM, a private `.uefi.fd` variables image. Omitted architecture inherits a
template's declared ISA. An explicit conflict fails before disk creation.
Cache pruning always protects backing images; MCP `unused_only` defaults to true
and explicitly disabling it is rejected. Raw imported disks without metadata require an explicit `--arch`; Nido cannot
reliably infer an instruction set from the QCOW2 container format.

A VM's architecture is immutable after creation. `start --arch` checks identity
or recovers a disk whose state is missing; it does not convert guest binaries.
Build recipes declare `arch: amd64` or `arch: arm64`, and must reference matching
installation media, drivers and automation. Changing the field alone does not
port an operating-system installer.

When upgrading from the earlier AMD64-only Nido:

1. Stop the VMs with the old binary and close other Nido CLI/TUI/MCP sessions.
2. Install the new binary, keeping the existing state directories and disks.
3. Run `nido migrate` once, then reopen clients. MCP clients can explicitly call
   `nido_system` with `action: migrate` instead.

The resumable conversion adds AMD64 declarations to old records and preserves
already declared ARM guests. It saves each original VM state beside it as
`.json.pre-architecture`, preserving unknown keys and permissions. It never
moves/copies disk payloads, rewrites backing chains, changes VM names, ports or
credentials. Existing image filenames remain valid disk paths and backing
references. New catalog acquisitions use ISA-qualified names; preserved old
cache files remain usable as explicit image paths or template sources.
Review and retain the state backups according to your normal backup policy.

`NIDO_HOME` selects a separate application-state root for containers and isolated
tests. It does not change the user's operating-system home. Keep custom image
and backup directories in that root's `config.env` when isolating a trial.

## Qualification boundaries

Automated contracts cover native selection, variant isolation, immutable VM
identity, template metadata and guest-aware acceleration. The isolated integration
suite exercises CLI, MCP and a real PTY selector. Setting `NIDO_TEST_QEMU_ARM=1`
adds a real ARM64 cloud guest, SSH `uname -m`, restart and template cloning.
An x86 host running ARM TCG does not qualify Raspberry Pi KVM, Apple Silicon HVF
or Windows ARM WHPX. Those physical hosts require their own lifecycle/workload
qualification before making performance or production-readiness claims.
